package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	defaultSessionTTL = 24 * time.Hour
	defaultPageSize   = 20
	maxPageSize       = 100
	minPasswordLength = 8
	maxPasswordLength = 1024

	reasonInvalidCredentials = "invalid_credentials"
	reasonAccountUnavailable = "account_unavailable"
	reasonDefaultRole        = "default_role"
	reasonUserLogout         = "user_logout"
	reasonUserDisabled       = "user_disabled"
	reasonPasswordReset      = "password_reset"
	reasonPasswordChanged    = "password_changed"
	reasonRoleChanged        = "role_changed"

	RoleStandard   = "standard"
	RoleAdmin      = "admin"
	RoleSuperAdmin = "super_admin"

	PermissionSystemAdmin = "system:admin"
)

type Clock func() time.Time

type IDGenerator func(prefix string) string

type TokenGenerator func() (string, error)

type Option func(*Service)

type Service struct {
	repo                Repository
	now                 Clock
	newID               IDGenerator
	newAccessToken      TokenGenerator
	tokenHashSecret     []byte
	tokenHashKeyVersion string
	sessionTTL          time.Duration
	defaultRoleCode     string
	logger              *slog.Logger
}

func New(repo Repository, opts ...Option) *Service {
	s := &Service{
		repo: repo,
		now: func() time.Time {
			return time.Now().UTC()
		},
		newID:               newID,
		newAccessToken:      newOpaqueAccessToken,
		tokenHashSecret:     []byte("auth-local-development-token-hash-secret"),
		tokenHashKeyVersion: TokenHashKeyVersionV1,
		sessionTTL:          defaultSessionTTL,
		defaultRoleCode:     DefaultRoleCode,
		logger:              slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func WithClock(now Clock) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

func WithIDGenerator(newID IDGenerator) Option {
	return func(s *Service) {
		if newID != nil {
			s.newID = newID
		}
	}
}

func WithTokenGenerator(newToken TokenGenerator) Option {
	return func(s *Service) {
		if newToken != nil {
			s.newAccessToken = newToken
		}
	}
}

func WithTokenHashSecret(secret []byte) Option {
	return func(s *Service) {
		s.tokenHashSecret = append([]byte(nil), secret...)
	}
}

func WithTokenHashKeyVersion(version string) Option {
	return func(s *Service) {
		if trimmed := strings.TrimSpace(version); trimmed != "" {
			s.tokenHashKeyVersion = trimmed
		}
	}
}

func WithSessionTTL(ttl time.Duration) Option {
	return func(s *Service) {
		if ttl > 0 {
			s.sessionTTL = ttl
		}
	}
}

func WithDefaultRoleCode(roleCode string) Option {
	return func(s *Service) {
		if trimmed := strings.TrimSpace(roleCode); trimmed != "" {
			s.defaultRoleCode = trimmed
		}
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.logger = logger
		}
	}
}

func (s *Service) CreateUser(ctx context.Context, reqCtx RequestContext, input CreateUserInput) (SessionResponse, error) {
	if err := s.validateReady(); err != nil {
		return SessionResponse{}, err
	}
	if err := validateInternalCaller(reqCtx); err != nil {
		return SessionResponse{}, err
	}

	username, password, err := normalizeCredentialsForCreate(input.Username, input.Password)
	if err != nil {
		return SessionResponse{}, err
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return SessionResponse{}, DependencyError("credential hashing failed", err)
	}

	now := s.now()
	user, err := s.repo.CreateUserWithCredential(ctx, CreateUserParams{
		ID:                        s.newID("usr"),
		Username:                  username,
		DisplayName:               username,
		Status:                    UserStatusActive,
		CreatedAt:                 now,
		PasswordCredentialID:      s.newID("cred"),
		PasswordHash:              passwordHash,
		PasswordHashAlg:           PasswordHashAlg,
		PasswordHashParamsVersion: PasswordHashParamsVersion,
		PasswordHashParamsJSON:    passwordHashParamsJSON(),
		DefaultRoleCode:           s.defaultRoleCode,
		RoleAssignmentID:          s.newID("urole"),
		AssignedBy:                callerService(reqCtx),
	})
	if err != nil {
		return SessionResponse{}, mapRepositoryError(err, "user not found")
	}

	userSummary := summaryFromRecord(user)
	s.recordSecurityEventBestEffort(ctx, reqCtx, SecurityEventParams{
		EventType:        SecurityEventUserCreated,
		UserID:           stringPtr(userSummary.ID),
		UsernameSnapshot: stringPtr(userSummary.Username),
		Status:           SecurityEventStatusSuccess,
	})
	if s.defaultRoleCode != "" {
		s.recordSecurityEventBestEffort(ctx, reqCtx, SecurityEventParams{
			EventType:        SecurityEventRoleAssigned,
			UserID:           stringPtr(userSummary.ID),
			UsernameSnapshot: stringPtr(userSummary.Username),
			Status:           SecurityEventStatusSuccess,
			ReasonCode:       stringPtr(reasonDefaultRole),
			MetadataJSON:     fmt.Sprintf(`{"role":%q}`, s.defaultRoleCode),
		})
	}

	session, err := s.createSessionForUser(ctx, reqCtx, userSummary)
	if err != nil {
		return SessionResponse{}, err
	}
	return SessionResponse{User: userSummary, Session: session}, nil
}

func (s *Service) CreateSession(ctx context.Context, reqCtx RequestContext, input CreateSessionInput) (SessionResponse, error) {
	if err := s.validateReady(); err != nil {
		return SessionResponse{}, err
	}
	if err := validateInternalCaller(reqCtx); err != nil {
		return SessionResponse{}, err
	}

	username, password, err := normalizeCredentialsForLogin(input.Username, input.Password)
	if err != nil {
		return SessionResponse{}, err
	}

	user, err := s.repo.FindUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			if eventErr := s.recordSessionFailure(ctx, reqCtx, nil, username, reasonInvalidCredentials); eventErr != nil {
				return SessionResponse{}, eventErr
			}
			return SessionResponse{}, invalidCredentialsError()
		}
		return SessionResponse{}, mapRepositoryError(err, "user not found")
	}

	if !userCanCreateSession(user.User, s.now()) {
		if eventErr := s.recordSessionFailure(ctx, reqCtx, &user.User, username, reasonAccountUnavailable); eventErr != nil {
			return SessionResponse{}, eventErr
		}
		return SessionResponse{}, invalidCredentialsError()
	}

	credential, err := s.repo.FindCredentialByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			if eventErr := s.recordSessionFailure(ctx, reqCtx, &user.User, username, reasonInvalidCredentials); eventErr != nil {
				return SessionResponse{}, eventErr
			}
			return SessionResponse{}, invalidCredentialsError()
		}
		return SessionResponse{}, mapRepositoryError(err, "credential not found")
	}
	if credential.PasswordHashAlg != PasswordHashAlg || credential.PasswordHashParamsVersion != PasswordHashParamsVersion {
		return SessionResponse{}, DependencyError("credential parameters are unsupported", nil)
	}
	ok, err := verifyPassword(password, credential.PasswordHash)
	if err != nil {
		return SessionResponse{}, DependencyError("credential verification failed", err)
	}
	if !ok {
		if eventErr := s.recordSessionFailure(ctx, reqCtx, &user.User, username, reasonInvalidCredentials); eventErr != nil {
			return SessionResponse{}, eventErr
		}
		return SessionResponse{}, invalidCredentialsError()
	}

	userSummary := summaryFromRecord(user)
	session, err := s.createSessionForUser(ctx, reqCtx, userSummary)
	if err != nil {
		return SessionResponse{}, err
	}
	return SessionResponse{User: userSummary, Session: session}, nil
}

func (s *Service) GetUser(ctx context.Context, reqCtx RequestContext, userID string) (UserRecord, error) {
	if err := s.validateReady(); err != nil {
		return UserRecord{}, err
	}
	if err := validateInternalCaller(reqCtx); err != nil {
		return UserRecord{}, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return UserRecord{}, ValidationError("request validation failed", map[string]string{"userId": "is required"})
	}
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return UserRecord{}, mapRepositoryError(err, "user not found")
	}
	return user, nil
}

func (s *Service) GetUserPermissions(ctx context.Context, reqCtx RequestContext, userID string) (UserPermissions, error) {
	user, err := s.GetUser(ctx, reqCtx, userID)
	if err != nil {
		return UserPermissions{}, err
	}
	return UserPermissions{
		UserID:      user.ID,
		Roles:       append([]string(nil), user.Roles...),
		Permissions: append([]string(nil), user.Permissions...),
		UpdatedAt:   user.UpdatedAt,
	}, nil
}

func (s *Service) ListManagedUsers(ctx context.Context, reqCtx RequestContext, input ListManagedUsersInput) (AdminUserList, error) {
	if err := s.validateReady(); err != nil {
		return AdminUserList{}, err
	}
	if err := validateInternalCaller(reqCtx); err != nil {
		return AdminUserList{}, err
	}
	actor, err := s.managementActor(ctx, reqCtx)
	if err != nil {
		return AdminUserList{}, err
	}
	username := strings.TrimSpace(input.Username)
	if len(username) > 128 {
		return AdminUserList{}, ValidationError("request validation failed", map[string]string{"username": "must be at most 128 characters"})
	}
	role := strings.TrimSpace(input.Role)
	if role != "" && !isManagedRole(role) {
		return AdminUserList{}, ValidationError("request validation failed", map[string]string{"role": "must be standard or admin"})
	}
	if role != "" && !actor.canManageRole(role) {
		return AdminUserList{}, ForbiddenError("forbidden")
	}
	status := strings.TrimSpace(input.Status)
	if status != "" && !isUserStatus(status) {
		return AdminUserList{}, ValidationError("request validation failed", map[string]string{"status": "must be active, disabled, or locked"})
	}
	page := input.Page
	if page <= 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	users, total, err := s.repo.ListManagedUsers(ctx, ListManagedUsersParams{
		ActorUserID:     actor.userID,
		ManageableRoles: actor.manageableRoles,
		ManagedRoles:    managedUserRoles(),
		Username:        username,
		Role:            role,
		Status:          status,
		Limit:           pageSize,
		Offset:          (page - 1) * pageSize,
	})
	if err != nil {
		return AdminUserList{}, mapRepositoryError(err, "user not found")
	}
	items := make([]AdminUserRecord, 0, len(users))
	for _, user := range users {
		items = append(items, adminUserFromRecord(user, actor))
	}
	return AdminUserList{
		Users: items,
		Page: PageInfo{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

func (s *Service) CreateAdminUser(ctx context.Context, reqCtx RequestContext, input CreateAdminUserInput) (AdminUserRecord, error) {
	if err := s.validateReady(); err != nil {
		return AdminUserRecord{}, err
	}
	if err := validateInternalCaller(reqCtx); err != nil {
		return AdminUserRecord{}, err
	}
	actor, err := s.managementActor(ctx, reqCtx)
	if err != nil {
		return AdminUserRecord{}, err
	}
	username := strings.TrimSpace(input.Username)
	if username == "" {
		return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"username": "is required"})
	}
	if len(username) > 128 {
		return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"username": "must be at most 128 characters"})
	}
	password, err := validateManagedPassword(input.TemporaryPassword)
	if err != nil {
		return AdminUserRecord{}, err
	}
	role := strings.TrimSpace(input.Role)
	if role == "" {
		return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"role": "is required"})
	}
	if !isManagedRole(role) || !actor.canManageRole(role) {
		return AdminUserRecord{}, ForbiddenError("forbidden")
	}
	displayName := ""
	if input.DisplayName != nil {
		displayName = strings.TrimSpace(*input.DisplayName)
	}
	if len(displayName) > 128 {
		return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"displayName": "must be at most 128 characters"})
	}
	if err := validateOptionalProfile("email", input.Email, 320); err != nil {
		return AdminUserRecord{}, err
	}
	if err := validateOptionalProfile("phone", input.Phone, 32); err != nil {
		return AdminUserRecord{}, err
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return AdminUserRecord{}, DependencyError("credential hashing failed", err)
	}
	now := s.now()
	user, err := s.repo.CreateUserWithCredential(ctx, CreateUserParams{
		ID:                        s.newID("usr"),
		Username:                  username,
		DisplayName:               displayName,
		Email:                     normalizedOptional(input.Email),
		Phone:                     normalizedOptional(input.Phone),
		Status:                    UserStatusActive,
		CreatedAt:                 now,
		PasswordCredentialID:      s.newID("cred"),
		PasswordHash:              passwordHash,
		PasswordHashAlg:           PasswordHashAlg,
		PasswordHashParamsVersion: PasswordHashParamsVersion,
		PasswordHashParamsJSON:    passwordHashParamsJSON(),
		MustChangePassword:        true,
		DefaultRoleCode:           role,
		RoleAssignmentID:          s.newID("urole"),
		AssignedBy:                actor.userID,
	})
	if err != nil {
		return AdminUserRecord{}, mapRepositoryError(err, "user not found")
	}
	s.recordSecurityEventBestEffort(ctx, reqCtx, SecurityEventParams{
		EventType:        SecurityEventUserCreated,
		UserID:           stringPtr(user.ID),
		UsernameSnapshot: stringPtr(user.Username),
		Status:           SecurityEventStatusSuccess,
		MetadataJSON:     fmt.Sprintf(`{"role":%q,"adminCreated":true}`, role),
	})
	return adminUserFromRecord(user, actor), nil
}

func (s *Service) UpdateManagedUser(ctx context.Context, reqCtx RequestContext, userID string, input UpdateAdminUserInput) (AdminUserRecord, error) {
	if err := s.validateReady(); err != nil {
		return AdminUserRecord{}, err
	}
	if err := validateInternalCaller(reqCtx); err != nil {
		return AdminUserRecord{}, err
	}
	actor, err := s.managementActor(ctx, reqCtx)
	if err != nil {
		return AdminUserRecord{}, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"userId": "is required"})
	}
	target, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return AdminUserRecord{}, mapRepositoryError(err, "user not found")
	}
	if userID == actor.userID && (input.Status.Set || input.Role.Set) {
		return AdminUserRecord{}, ForbiddenError("forbidden")
	}
	if !actor.canManageRecord(target) {
		return AdminUserRecord{}, NotFoundError("user not found")
	}
	if !input.hasAny() {
		return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"body": "must include at least one editable field"})
	}
	needsProfileUpdate := input.DisplayName.Set || input.Email.Set || input.Phone.Set
	displayName := target.DisplayName
	if input.DisplayName.Set {
		if input.DisplayName.Value == nil {
			return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"displayName": "must not be null"})
		}
		displayName = strings.TrimSpace(*input.DisplayName.Value)
	}
	email := target.Email
	if input.Email.Set {
		email = normalizedOptional(input.Email.Value)
	}
	phone := target.Phone
	if input.Phone.Set {
		phone = normalizedOptional(input.Phone.Value)
	}
	if needsProfileUpdate {
		if len(displayName) > 128 {
			return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"displayName": "must be at most 128 characters"})
		}
		if err := validateOptionalProfile("email", email, 320); err != nil {
			return AdminUserRecord{}, err
		}
		if err := validateOptionalProfile("phone", phone, 32); err != nil {
			return AdminUserRecord{}, err
		}
	}
	nextStatus := target.Status
	if input.Status.Set {
		if input.Status.Value == nil {
			return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"status": "must not be null"})
		}
		status := strings.TrimSpace(*input.Status.Value)
		if status != UserStatusActive && status != UserStatusDisabled {
			return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"status": "must be active or disabled"})
		}
		nextStatus = status
	}
	nextRole := ""
	if input.Role.Set {
		if input.Role.Value == nil {
			return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"role": "must not be null"})
		}
		role := strings.TrimSpace(*input.Role.Value)
		if !isManagedRole(role) {
			return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"role": "must be standard or admin"})
		}
		if userID == actor.userID {
			return AdminUserRecord{}, ForbiddenError("forbidden")
		}
		if !actor.canManageRole(role) {
			return AdminUserRecord{}, ForbiddenError("forbidden")
		}
		nextRole = role
	}

	changedSecurityState := false
	updated := target
	if needsProfileUpdate {
		updated, err = s.repo.UpdateUserProfile(ctx, UpdateUserProfileParams{
			UserID:      userID,
			DisplayName: displayName,
			Email:       email,
			Phone:       phone,
			UpdatedAt:   s.now(),
		})
		if err != nil {
			return AdminUserRecord{}, mapRepositoryError(err, "user not found")
		}
	}
	if input.Status.Set && updated.Status != nextStatus {
		updated, err = s.repo.UpdateUserStatus(ctx, UpdateUserStatusParams{UserID: userID, Status: nextStatus, UpdatedAt: s.now()})
		if err != nil {
			return AdminUserRecord{}, mapRepositoryError(err, "user not found")
		}
		if nextStatus == UserStatusDisabled {
			changedSecurityState = true
		}
	}
	if input.Role.Set {
		if !userHasRole(updated, nextRole) || len(managedRolesForRecord(updated)) != 1 {
			updated, err = s.repo.ReplaceUserRole(ctx, ReplaceUserRoleParams{
				UserID:           userID,
				RoleCode:         nextRole,
				ManagedRoleCodes: []string{RoleStandard, RoleAdmin},
				AssignmentID:     s.newID("urole"),
				AssignedBy:       actor.userID,
				AssignedAt:       s.now(),
			})
			if err != nil {
				return AdminUserRecord{}, mapRepositoryError(err, "user not found")
			}
			changedSecurityState = true
			s.recordSecurityEventBestEffort(ctx, reqCtx, SecurityEventParams{
				EventType:    SecurityEventRoleAssigned,
				UserID:       stringPtr(userID),
				Status:       SecurityEventStatusSuccess,
				ReasonCode:   stringPtr(reasonRoleChanged),
				MetadataJSON: fmt.Sprintf(`{"role":%q}`, nextRole),
			})
		}
	}
	if changedSecurityState {
		_ = s.revokeUserSessionsBestEffort(ctx, reqCtx, userID, reasonRoleOrStatusChange(updated.Status))
	}
	return adminUserFromRecord(updated, actor), nil
}

func (s *Service) ResetManagedUserPassword(ctx context.Context, reqCtx RequestContext, userID string, input ResetAdminPasswordInput) (AdminUserRecord, error) {
	if err := s.validateReady(); err != nil {
		return AdminUserRecord{}, err
	}
	if err := validateInternalCaller(reqCtx); err != nil {
		return AdminUserRecord{}, err
	}
	actor, err := s.managementActor(ctx, reqCtx)
	if err != nil {
		return AdminUserRecord{}, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return AdminUserRecord{}, ValidationError("request validation failed", map[string]string{"userId": "is required"})
	}
	if userID == actor.userID {
		return AdminUserRecord{}, ForbiddenError("forbidden")
	}
	target, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return AdminUserRecord{}, mapRepositoryError(err, "user not found")
	}
	if !actor.canManageRecord(target) {
		return AdminUserRecord{}, NotFoundError("user not found")
	}
	password, err := validateManagedPassword(input.TemporaryPassword)
	if err != nil {
		return AdminUserRecord{}, err
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return AdminUserRecord{}, DependencyError("credential hashing failed", err)
	}
	if _, err := s.repo.UpdatePassword(ctx, UpdatePasswordParams{
		UserID:                    userID,
		PasswordHash:              passwordHash,
		PasswordHashAlg:           PasswordHashAlg,
		PasswordHashParamsVersion: PasswordHashParamsVersion,
		PasswordHashParamsJSON:    passwordHashParamsJSON(),
		MustChangePassword:        true,
		ChangedAt:                 s.now(),
	}); err != nil {
		return AdminUserRecord{}, mapRepositoryError(err, "credential not found")
	}
	updated, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return AdminUserRecord{}, mapRepositoryError(err, "user not found")
	}
	s.recordSecurityEventBestEffort(ctx, reqCtx, SecurityEventParams{
		EventType:        SecurityEventPasswordReset,
		UserID:           stringPtr(updated.ID),
		UsernameSnapshot: stringPtr(updated.Username),
		Status:           SecurityEventStatusSuccess,
		ReasonCode:       stringPtr(reasonPasswordReset),
		MetadataJSON:     fmt.Sprintf(`{"actorUserId":%q}`, actor.userID),
	})
	_ = s.revokeUserSessionsBestEffort(ctx, reqCtx, userID, reasonPasswordReset)
	return adminUserFromRecord(updated, actor), nil
}

func (s *Service) UpdateProfile(ctx context.Context, reqCtx RequestContext, userID string, input UpdateProfileInput) (UserRecord, error) {
	if err := s.validateReady(); err != nil {
		return UserRecord{}, err
	}
	if err := validateGatewayCaller(reqCtx); err != nil {
		return UserRecord{}, err
	}
	if err := validateSelfActor(reqCtx, userID); err != nil {
		return UserRecord{}, err
	}
	current, err := s.repo.FindUserByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return UserRecord{}, mapRepositoryError(err, "user not found")
	}
	if !input.hasAny() {
		return UserRecord{}, ValidationError("request validation failed", map[string]string{"body": "must include at least one editable field"})
	}
	displayName := current.DisplayName
	if input.DisplayName.Set {
		if input.DisplayName.Value == nil {
			return UserRecord{}, ValidationError("request validation failed", map[string]string{"displayName": "must not be null"})
		}
		displayName = ""
		displayName = strings.TrimSpace(*input.DisplayName.Value)
	}
	if len(displayName) > 128 {
		return UserRecord{}, ValidationError("request validation failed", map[string]string{"displayName": "must be at most 128 characters"})
	}
	email := current.Email
	if input.Email.Set {
		email = normalizedOptional(input.Email.Value)
	}
	phone := current.Phone
	if input.Phone.Set {
		phone = normalizedOptional(input.Phone.Value)
	}
	if err := validateOptionalProfile("email", email, 320); err != nil {
		return UserRecord{}, err
	}
	if err := validateOptionalProfile("phone", phone, 32); err != nil {
		return UserRecord{}, err
	}
	updated, err := s.repo.UpdateUserProfile(ctx, UpdateUserProfileParams{
		UserID:      strings.TrimSpace(userID),
		DisplayName: displayName,
		Email:       email,
		Phone:       phone,
		UpdatedAt:   s.now(),
	})
	if err != nil {
		return UserRecord{}, mapRepositoryError(err, "user not found")
	}
	return updated, nil
}

func (s *Service) ChangeRequiredPassword(ctx context.Context, reqCtx RequestContext, userID string, input ChangePasswordInput) (UserRecord, error) {
	if err := s.validateReady(); err != nil {
		return UserRecord{}, err
	}
	if err := validateGatewayCaller(reqCtx); err != nil {
		return UserRecord{}, err
	}
	if err := validateSelfActor(reqCtx, userID); err != nil {
		return UserRecord{}, err
	}
	newPassword, err := validatePasswordChangeInput(input)
	if err != nil {
		return UserRecord{}, err
	}
	user, err := s.repo.FindUserByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return UserRecord{}, mapRepositoryError(err, "user not found")
	}
	if !user.MustChangePassword {
		return UserRecord{}, ConflictError("password change is not required", nil)
	}
	credential, err := s.repo.FindCredentialByUserID(ctx, user.ID)
	if err != nil {
		return UserRecord{}, mapRepositoryError(err, "credential not found")
	}
	if credential.PasswordHashAlg != PasswordHashAlg || credential.PasswordHashParamsVersion != PasswordHashParamsVersion {
		return UserRecord{}, DependencyError("credential parameters are unsupported", nil)
	}
	ok, err := verifyPassword(input.CurrentPassword, credential.PasswordHash)
	if err != nil {
		return UserRecord{}, DependencyError("credential verification failed", err)
	}
	if !ok {
		return UserRecord{}, NewError(CodeUnauthorized, "invalid current password", nil)
	}
	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return UserRecord{}, DependencyError("credential hashing failed", err)
	}
	if _, err := s.repo.UpdatePassword(ctx, UpdatePasswordParams{
		UserID:                    user.ID,
		PasswordHash:              passwordHash,
		PasswordHashAlg:           PasswordHashAlg,
		PasswordHashParamsVersion: PasswordHashParamsVersion,
		PasswordHashParamsJSON:    passwordHashParamsJSON(),
		MustChangePassword:        false,
		ChangedAt:                 s.now(),
	}); err != nil {
		return UserRecord{}, mapRepositoryError(err, "credential not found")
	}
	updated, err := s.repo.FindUserByID(ctx, user.ID)
	if err != nil {
		return UserRecord{}, mapRepositoryError(err, "user not found")
	}
	s.recordSecurityEventBestEffort(ctx, reqCtx, SecurityEventParams{
		EventType:        SecurityEventPasswordChanged,
		UserID:           stringPtr(updated.ID),
		UsernameSnapshot: stringPtr(updated.Username),
		Status:           SecurityEventStatusSuccess,
		ReasonCode:       stringPtr(reasonPasswordChanged),
	})
	return updated, nil
}

func (s *Service) GetSession(ctx context.Context, reqCtx RequestContext, sessionID string) (SessionIdentity, error) {
	if err := s.validateReady(); err != nil {
		return SessionIdentity{}, err
	}
	if err := validateInternalCaller(reqCtx); err != nil {
		return SessionIdentity{}, err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return SessionIdentity{}, ValidationError("request validation failed", map[string]string{"sessionId": "is required"})
	}
	identity, err := s.repo.FindSessionByID(ctx, sessionID)
	if err != nil {
		return SessionIdentity{}, mapRepositoryError(err, "session not found")
	}
	if !isActiveSessionIdentity(identity, s.now()) {
		return SessionIdentity{}, NotFoundError("session not found")
	}
	return identity, nil
}

func (s *Service) GetSessionByAccessToken(ctx context.Context, reqCtx RequestContext, accessToken string) (SessionIdentity, error) {
	if err := s.validateReady(); err != nil {
		return SessionIdentity{}, err
	}
	if err := validateInternalCaller(reqCtx); err != nil {
		return SessionIdentity{}, err
	}
	tokenHash, err := hashAccessToken(accessToken, s.tokenHashSecret, s.tokenHashKeyVersion)
	if err != nil {
		return SessionIdentity{}, UnauthorizedError()
	}
	identity, err := s.repo.FindActiveSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return SessionIdentity{}, UnauthorizedError()
		}
		return SessionIdentity{}, mapRepositoryError(err, "session not found")
	}
	return identity, nil
}

func (s *Service) RevokeSession(ctx context.Context, reqCtx RequestContext, sessionID string, reason string) error {
	if err := s.validateReady(); err != nil {
		return err
	}
	if err := validateInternalCaller(reqCtx); err != nil {
		return err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ValidationError("request validation failed", map[string]string{"sessionId": "is required"})
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = reasonUserLogout
	}

	session, err := s.repo.RevokeSession(ctx, RevokeSessionParams{
		SessionID: sessionID,
		Reason:    reason,
		RequestID: optionalString(reqCtx.RequestID),
		RevokedAt: s.now(),
	})
	if err != nil {
		return mapRepositoryError(err, "session not found")
	}
	s.recordSecurityEventBestEffort(ctx, reqCtx, SecurityEventParams{
		EventType:    SecurityEventSessionRevoked,
		UserID:       stringPtr(session.UserID),
		SessionID:    stringPtr(session.ID),
		Status:       SecurityEventStatusSuccess,
		ReasonCode:   stringPtr(reason),
		MetadataJSON: "{}",
	})
	return nil
}

func (s *Service) createSessionForUser(ctx context.Context, reqCtx RequestContext, user UserSummary) (SessionSummary, error) {
	accessToken, err := s.newAccessToken()
	if err != nil {
		return SessionSummary{}, DependencyError("access token generation failed", err)
	}
	tokenHash, err := hashAccessToken(accessToken, s.tokenHashSecret, s.tokenHashKeyVersion)
	if err != nil {
		return SessionSummary{}, DependencyError("access token hashing failed", err)
	}
	issuedAt := s.now()
	identity, err := s.repo.CreateSession(ctx, CreateSessionParams{
		ID:                        s.newID("sess"),
		UserID:                    user.ID,
		AccessTokenHash:           tokenHash,
		AccessTokenHashAlg:        TokenHashAlg,
		AccessTokenHashKeyVersion: s.tokenHashKeyVersion,
		IssuedAt:                  issuedAt,
		ExpiresAt:                 issuedAt.Add(s.sessionTTL),
		ClientIP:                  optionalString(clientIP(reqCtx)),
		UserAgent:                 optionalString(reqCtx.UserAgent),
		RequestID:                 optionalString(reqCtx.RequestID),
	})
	if err != nil {
		return SessionSummary{}, mapRepositoryError(err, "session not found")
	}
	s.recordSecurityEventBestEffort(ctx, reqCtx, SecurityEventParams{
		EventType:        SecurityEventSessionCreated,
		UserID:           stringPtr(user.ID),
		SessionID:        stringPtr(identity.Session.ID),
		UsernameSnapshot: stringPtr(user.Username),
		Status:           SecurityEventStatusSuccess,
	})
	return SessionSummary{
		SessionID:   identity.Session.ID,
		AccessToken: accessToken,
		TokenType:   identity.Session.TokenType,
		ExpiresAt:   identity.Session.ExpiresAt,
	}, nil
}

func (s *Service) recordSessionFailure(ctx context.Context, reqCtx RequestContext, user *User, username string, reason string) error {
	var userID *string
	if user != nil {
		userID = stringPtr(user.ID)
	}
	return s.recordSecurityEvent(ctx, reqCtx, SecurityEventParams{
		EventType:        SecurityEventSessionCreateFailed,
		UserID:           userID,
		UsernameSnapshot: optionalString(username),
		Status:           SecurityEventStatusFailed,
		ReasonCode:       stringPtr(reason),
	})
}

func (s *Service) recordSecurityEvent(ctx context.Context, reqCtx RequestContext, params SecurityEventParams) error {
	params.ID = s.newID("sevt")
	params.RequestID = optionalString(reqCtx.RequestID)
	params.ClientIP = optionalString(clientIP(reqCtx))
	params.UserAgent = optionalString(reqCtx.UserAgent)
	params.CallerService = optionalString(reqCtx.CallerService)
	params.CreatedAt = s.now()
	if strings.TrimSpace(params.MetadataJSON) == "" {
		params.MetadataJSON = "{}"
	}
	if err := s.repo.RecordSecurityEvent(ctx, params); err != nil {
		return DependencyError("security event write failed", err)
	}
	return nil
}

func (s *Service) recordSecurityEventBestEffort(ctx context.Context, reqCtx RequestContext, params SecurityEventParams) {
	eventType := params.EventType
	if err := s.recordSecurityEvent(ctx, reqCtx, params); err != nil {
		s.logSecurityEventFailure(ctx, reqCtx, eventType, err)
	}
}

func (s *Service) logSecurityEventFailure(ctx context.Context, reqCtx RequestContext, eventType string, err error) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.WarnContext(ctx, "security event write failed",
		"service", "auth",
		"request_id", strings.TrimSpace(reqCtx.RequestID),
		"operation", "record_security_event",
		"event_type", strings.TrimSpace(eventType),
		"status", "failed",
		"error", err,
	)
}

func (s *Service) validateReady() error {
	if s == nil || s.repo == nil {
		return DependencyError("auth repository is not configured", nil)
	}
	if len(s.tokenHashSecret) == 0 {
		return DependencyError("token hash secret is not configured", nil)
	}
	return nil
}

func normalizeCredentialsForLogin(username string, password string) (string, string, error) {
	fields := map[string]string{}
	username = strings.TrimSpace(username)
	if username == "" {
		fields["username"] = "is required"
	}
	if len(username) > 128 {
		fields["username"] = "must be at most 128 characters"
	}
	if password == "" {
		fields["password"] = "is required"
	}
	if len(password) > maxPasswordLength {
		fields["password"] = fmt.Sprintf("must be at most %d characters", maxPasswordLength)
	}
	if len(fields) > 0 {
		return "", "", ValidationError("request validation failed", fields)
	}
	return username, password, nil
}

func normalizeCredentialsForCreate(username string, password string) (string, string, error) {
	username, password, err := normalizeCredentialsForLogin(username, password)
	if err != nil {
		return "", "", err
	}
	if len(password) < minPasswordLength {
		return "", "", ValidationError("request validation failed", map[string]string{"password": fmt.Sprintf("must be at least %d characters", minPasswordLength)})
	}
	return username, password, nil
}

func validateManagedPassword(password string) (string, error) {
	if password == "" {
		return "", ValidationError("request validation failed", map[string]string{"temporaryPassword": "is required"})
	}
	if len(password) < minPasswordLength {
		return "", ValidationError("request validation failed", map[string]string{"temporaryPassword": fmt.Sprintf("must be at least %d characters", minPasswordLength)})
	}
	if len(password) > maxPasswordLength {
		return "", ValidationError("request validation failed", map[string]string{"temporaryPassword": fmt.Sprintf("must be at most %d characters", maxPasswordLength)})
	}
	return password, nil
}

func validatePasswordChangeInput(input ChangePasswordInput) (string, error) {
	fields := map[string]string{}
	if input.CurrentPassword == "" {
		fields["currentPassword"] = "is required"
	} else if len(input.CurrentPassword) > maxPasswordLength {
		fields["currentPassword"] = fmt.Sprintf("must be at most %d characters", maxPasswordLength)
	}
	if input.NewPassword == "" {
		fields["newPassword"] = "is required"
	} else if len(input.NewPassword) < minPasswordLength {
		fields["newPassword"] = fmt.Sprintf("must be at least %d characters", minPasswordLength)
	} else if len(input.NewPassword) > maxPasswordLength {
		fields["newPassword"] = fmt.Sprintf("must be at most %d characters", maxPasswordLength)
	}
	if input.NewPasswordConfirmation == "" {
		fields["newPasswordConfirmation"] = "is required"
	} else if input.NewPasswordConfirmation != input.NewPassword {
		fields["newPasswordConfirmation"] = "must match newPassword"
	}
	if len(fields) > 0 {
		return "", ValidationError("request validation failed", fields)
	}
	return input.NewPassword, nil
}

func (input UpdateAdminUserInput) hasAny() bool {
	return input.DisplayName.Set || input.Email.Set || input.Phone.Set || input.Status.Set || input.Role.Set
}

func (input UpdateProfileInput) hasAny() bool {
	return input.DisplayName.Set || input.Email.Set || input.Phone.Set
}

type managementActor struct {
	userID          string
	roles           []string
	manageableRoles []string
}

func (s *Service) managementActor(ctx context.Context, reqCtx RequestContext) (managementActor, error) {
	if !strings.EqualFold(strings.TrimSpace(reqCtx.CallerService), "gateway") {
		return managementActor{}, UnauthorizedError()
	}
	userID := strings.TrimSpace(reqCtx.ActorUserID)
	if userID == "" {
		return managementActor{}, UnauthorizedError()
	}
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return managementActor{}, mapRepositoryError(err, "user not found")
	}
	if user.Status != UserStatusActive {
		return managementActor{}, ForbiddenError("forbidden")
	}
	actor := managementActor{
		userID: userID,
		roles:  normalizedStrings(user.Roles),
	}
	if hasStringFold(actor.roles, RoleSuperAdmin) {
		actor.manageableRoles = []string{RoleStandard, RoleAdmin}
		return actor, nil
	}
	if hasStringFold(actor.roles, RoleAdmin) {
		actor.manageableRoles = []string{RoleStandard}
		return actor, nil
	}
	return managementActor{}, ForbiddenError("forbidden")
}

func (actor managementActor) canManageRole(role string) bool {
	return hasStringFold(actor.manageableRoles, role)
}

func (actor managementActor) canManageRecord(user UserRecord) bool {
	roles := managedRolesForRecord(user)
	if len(roles) == 0 {
		return false
	}
	for _, role := range roles {
		if !actor.canManageRole(role) {
			return false
		}
	}
	return true
}

func managedRolesForRecord(user UserRecord) []string {
	roles := make([]string, 0, len(managedUserRoles()))
	for _, role := range user.Roles {
		role = strings.TrimSpace(role)
		if hasStringFold(managedUserRoles(), role) {
			roles = append(roles, role)
		}
	}
	return roles
}

func managedUserRoles() []string {
	return []string{RoleStandard, RoleAdmin, RoleSuperAdmin}
}

func userHasRole(user UserRecord, role string) bool {
	return hasStringFold(user.Roles, role)
}

func adminUserFromRecord(user UserRecord, actor managementActor) AdminUserRecord {
	manageableRoles := []string{}
	if actor.canManageRecord(user) {
		manageableRoles = append(manageableRoles, actor.manageableRoles...)
	}
	self := strings.TrimSpace(user.ID) == actor.userID
	actions := AdminUserActions{
		CanDisable:       !self && actor.canManageRecord(user) && user.Status == UserStatusActive,
		CanEnable:        !self && actor.canManageRecord(user) && user.Status == UserStatusDisabled,
		CanResetPassword: !self && actor.canManageRecord(user),
		CanChangeRole:    !self && actor.canManageRecord(user),
	}
	return AdminUserRecord{
		UserRecord:      user,
		ManageableRoles: manageableRoles,
		Actions:         actions,
	}
}

func validateSelfActor(reqCtx RequestContext, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ValidationError("request validation failed", map[string]string{"userId": "is required"})
	}
	actorID := strings.TrimSpace(reqCtx.ActorUserID)
	if actorID == "" {
		return UnauthorizedError()
	}
	if actorID != userID {
		return ForbiddenError("forbidden")
	}
	return nil
}

func validateOptionalProfile(field string, value *string, max int) error {
	if value == nil {
		return nil
	}
	if len(*value) > max {
		return ValidationError("request validation failed", map[string]string{field: fmt.Sprintf("must be at most %d characters", max)})
	}
	return nil
}

func normalizedOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func isManagedRole(role string) bool {
	role = strings.TrimSpace(role)
	return role == RoleStandard || role == RoleAdmin
}

func isUserStatus(status string) bool {
	status = strings.TrimSpace(status)
	return status == UserStatusActive || status == UserStatusDisabled || status == UserStatusLocked
}

func hasStringFold(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func isActiveSessionIdentity(identity SessionIdentity, now time.Time) bool {
	session := identity.Session
	if session.Status != SessionStatusActive {
		return false
	}
	if !session.ExpiresAt.After(now) {
		return false
	}
	return session.RevokedAt == nil
}

func normalizedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func reasonRoleOrStatusChange(status string) string {
	if status == UserStatusDisabled {
		return reasonUserDisabled
	}
	return reasonRoleChanged
}

func (s *Service) revokeUserSessionsBestEffort(ctx context.Context, reqCtx RequestContext, userID string, reason string) error {
	if reason == "" {
		reason = reasonRoleChanged
	}
	_, err := s.repo.RevokeUserSessions(ctx, RevokeUserSessionsParams{
		UserID:    userID,
		Reason:    reason,
		RequestID: optionalString(reqCtx.RequestID),
		RevokedAt: s.now(),
	})
	if err != nil {
		s.logSecurityEventFailure(ctx, reqCtx, SecurityEventSessionRevoked, err)
	}
	return err
}

func validateInternalCaller(reqCtx RequestContext) error {
	if strings.TrimSpace(reqCtx.CallerService) == "" {
		return UnauthorizedError()
	}
	return nil
}

func validateGatewayCaller(reqCtx RequestContext) error {
	if err := validateInternalCaller(reqCtx); err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(reqCtx.CallerService), "gateway") {
		return ForbiddenError("forbidden")
	}
	return nil
}

func userCanCreateSession(user User, now time.Time) bool {
	if user.Status != UserStatusActive {
		return false
	}
	return user.LockedUntil == nil || !user.LockedUntil.After(now)
}

func summaryFromRecord(user UserRecord) UserSummary {
	return UserSummary{
		ID:                 user.ID,
		Username:           user.Username,
		DisplayName:        user.DisplayName,
		Email:              cloneStringPtr(user.Email),
		Phone:              cloneStringPtr(user.Phone),
		Status:             user.Status,
		MustChangePassword: user.MustChangePassword,
		Roles:              append([]string(nil), user.Roles...),
		Permissions:        append([]string(nil), user.Permissions...),
	}
}

func invalidCredentialsError() error {
	return NewError(CodeUnauthorized, "invalid username or password", nil)
}

func mapRepositoryError(err error, notFoundMessage string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return NotFoundError(notFoundMessage)
	}
	if errors.Is(err, ErrConflict) {
		return ConflictError("resource already exists", err)
	}
	if _, ok := Classify(err); ok {
		return err
	}
	return DependencyError("repository operation failed", err)
}

func callerService(reqCtx RequestContext) string {
	caller := strings.TrimSpace(reqCtx.CallerService)
	if caller == "" {
		return "gateway"
	}
	return caller
}

func clientIP(reqCtx RequestContext) string {
	if trimmed := strings.TrimSpace(reqCtx.ClientIP); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(reqCtx.ForwardedFor)
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func newID(prefix string) string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return prefix + "_" + hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}
