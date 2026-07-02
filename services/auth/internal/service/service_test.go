package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPasswordHashUsesArgon2idV1PHC(t *testing.T) {
	hash, err := hashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hashPassword() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("hash = %q", hash)
	}
	ok, err := verifyPassword("correct horse battery staple", hash)
	if err != nil {
		t.Fatalf("verifyPassword() error = %v", err)
	}
	if !ok {
		t.Fatalf("verifyPassword() = false")
	}
	ok, err = verifyPassword("wrong", hash)
	if err != nil {
		t.Fatalf("verifyPassword(wrong) error = %v", err)
	}
	if ok {
		t.Fatalf("verifyPassword(wrong) = true")
	}
}

func TestLocalDemoSeedPasswordHashMatchesDocumentedPassword(t *testing.T) {
	const localDemoHash = "$argon2id$v=19$m=65536,t=3,p=2$bG9jYWwtZGVtby1zYWx0IQ$tESTl/LqUlaDlE8hP4+CNLG5go/+X2xvYXBdqk+4eOI"

	ok, err := verifyPassword("LocalDemoAdmin#12345", localDemoHash)
	if err != nil {
		t.Fatalf("verifyPassword() error = %v", err)
	}
	if !ok {
		t.Fatalf("verifyPassword() = false")
	}
}

func TestAccessTokenHashIsVersionedHMAC(t *testing.T) {
	hash, err := hashAccessToken("atk_v1_example", []byte("secret"), "v1")
	if err != nil {
		t.Fatalf("hashAccessToken() error = %v", err)
	}
	if !strings.HasPrefix(hash, "hmac-sha256:v1:") {
		t.Fatalf("hash = %q", hash)
	}
	if strings.Contains(hash, "atk_v1_example") {
		t.Fatalf("hash leaks raw token: %q", hash)
	}
	again, err := hashAccessToken("atk_v1_example", []byte("secret"), "v1")
	if err != nil {
		t.Fatalf("hashAccessToken() second error = %v", err)
	}
	if hash != again {
		t.Fatalf("hash is not deterministic: %q != %q", hash, again)
	}
}

func TestCreateSessionRejectsWrongPasswordAndRecordsFailure(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_fixed")

	_, err := svc.CreateSession(context.Background(), testRequestContext(), CreateSessionInput{
		Username: "alice",
		Password: "wrong-password",
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeUnauthorized {
		t.Fatalf("code = %s", appErr.Code)
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("sessions = %+v", repo.sessions)
	}
	if !repo.hasEvent(SecurityEventSessionCreateFailed, SecurityEventStatusFailed, reasonInvalidCredentials) {
		t.Fatalf("events = %+v", repo.events)
	}
}

func TestCreateUserRejectsDuplicateUsername(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_fixed")

	_, err := svc.CreateUser(context.Background(), testRequestContext(), CreateUserInput{
		Username: "alice",
		Password: "new-password",
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeConflict {
		t.Fatalf("code = %s", appErr.Code)
	}
}

func TestCreateUserReturnsTokenButPersistsOnlyHash(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")

	result, err := svc.CreateUser(context.Background(), testRequestContext(), CreateUserInput{
		Username: "bob",
		Password: "bob-password",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if result.Session.AccessToken != "atk_v1_created" {
		t.Fatalf("access token = %q", result.Session.AccessToken)
	}
	if result.Session.SessionID == "" || result.Session.TokenType != TokenTypeBearer {
		t.Fatalf("session = %+v", result.Session)
	}
	if got, want := strings.Join(result.User.Roles, ","), "standard"; got != want {
		t.Fatalf("roles = %q", got)
	}
	if result.User.MustChangePassword {
		t.Fatalf("public registration must not require password change")
	}
	stored := repo.sessions[result.Session.SessionID]
	if stored.AccessTokenHash == "" || !strings.HasPrefix(stored.AccessTokenHash, "hmac-sha256:v1:") {
		t.Fatalf("stored hash = %q", stored.AccessTokenHash)
	}
	if strings.Contains(stored.AccessTokenHash, result.Session.AccessToken) {
		t.Fatalf("stored hash leaks token: %q", stored.AccessTokenHash)
	}
	if !repo.hasEvent(SecurityEventUserCreated, SecurityEventStatusSuccess, "") ||
		!repo.hasEvent(SecurityEventRoleAssigned, SecurityEventStatusSuccess, reasonDefaultRole) ||
		!repo.hasEvent(SecurityEventSessionCreated, SecurityEventStatusSuccess, "") {
		t.Fatalf("events = %+v", repo.events)
	}
}

func TestCreateUserRejectsShortPassword(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")

	_, err := svc.CreateUser(context.Background(), testRequestContext(), CreateUserInput{
		Username: "bob",
		Password: "short",
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeValidation {
		t.Fatalf("code = %s", appErr.Code)
	}
}

func TestAdminCanCreateStandardUserWithMustChangePassword(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")

	user, err := svc.CreateAdminUser(context.Background(), adminRequestContext(), CreateAdminUserInput{
		Username:          "bob",
		TemporaryPassword: "temporary-password",
		Role:              RoleStandard,
	})
	if err != nil {
		t.Fatalf("CreateAdminUser() error = %v", err)
	}
	if user.Username != "bob" || !user.MustChangePassword || !user.Actions.CanResetPassword {
		t.Fatalf("user = %+v", user)
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("admin create should not create session: %+v", repo.sessions)
	}
}

func TestAdminCreateUserUsesTemporaryPasswordValidationField(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")

	_, err := svc.CreateAdminUser(context.Background(), adminRequestContext(), CreateAdminUserInput{
		Username:          "bob",
		TemporaryPassword: "short",
		Role:              RoleStandard,
	})
	appErr := requireAppError(t, err)
	if appErr.Code != CodeValidation || appErr.Fields["temporaryPassword"] == "" {
		t.Fatalf("error = %+v", appErr)
	}
	if appErr.Fields["password"] != "" {
		t.Fatalf("unexpected password field error = %+v", appErr.Fields)
	}
}

func TestAdminCannotCreateAdminUser(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")

	_, err := svc.CreateAdminUser(context.Background(), adminRequestContext(), CreateAdminUserInput{
		Username:          "ops",
		TemporaryPassword: "temporary-password",
		Role:              RoleAdmin,
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeForbidden {
		t.Fatalf("code = %s", appErr.Code)
	}
}

func TestManagementActorIgnoresSpoofedRoleHeaders(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")
	ctx := testRequestContext()
	ctx.ActorUserID = "usr_alice"
	ctx.ActorRoles = []string{RoleSuperAdmin}

	_, err := svc.CreateAdminUser(context.Background(), ctx, CreateAdminUserInput{
		Username:          "ops",
		TemporaryPassword: "temporary-password",
		Role:              RoleAdmin,
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeForbidden {
		t.Fatalf("code = %s", appErr.Code)
	}
	if _, ok := repo.usersByUsername["ops"]; ok {
		t.Fatalf("spoofed management request created user")
	}
}

func TestManagementActorRejectsDisabledActor(t *testing.T) {
	repo := newFakeRepository(t)
	actor := repo.usersByID["usr_admin"]
	actor.Status = UserStatusDisabled
	repo.usersByID[actor.ID] = actor
	repo.usersByUsername[actor.Username] = actor
	svc := newTestService(repo, "atk_v1_created")

	_, err := svc.ListManagedUsers(context.Background(), adminRequestContext(), ListManagedUsersInput{})
	if appErr := requireAppError(t, err); appErr.Code != CodeForbidden {
		t.Fatalf("code = %s", appErr.Code)
	}
}

func TestUpdateManagedUserRejectsRoleBeforeProfileWrite(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")
	changedName := "Changed"
	targetID := "usr_alice"

	_, err := svc.UpdateManagedUser(context.Background(), adminRequestContext(), targetID, UpdateAdminUserInput{
		DisplayName: OptionalStringField{Set: true, Value: &changedName},
		Role:        OptionalStringField{Set: true, Value: stringPtr(RoleAdmin)},
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeForbidden {
		t.Fatalf("code = %s", appErr.Code)
	}
	if got := repo.usersByID[targetID].DisplayName; got != "" {
		t.Fatalf("display name was partially written: %q", got)
	}
}

func TestUpdateManagedUserValidatesStatusBeforeProfileWrite(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")
	changedName := "Changed"
	invalidStatus := "archived"
	targetID := "usr_alice"

	_, err := svc.UpdateManagedUser(context.Background(), adminRequestContext(), targetID, UpdateAdminUserInput{
		DisplayName: OptionalStringField{Set: true, Value: &changedName},
		Status:      OptionalStringField{Set: true, Value: &invalidStatus},
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeValidation {
		t.Fatalf("code = %s", appErr.Code)
	}
	if got := repo.usersByID[targetID].DisplayName; got != "" {
		t.Fatalf("display name was partially written: %q", got)
	}
}

func TestAdminSystemPermissionDoesNotGrantSuperAdminUserManagement(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")
	ctx := adminRequestContext()
	ctx.ActorPerms = []string{PermissionSystemAdmin}

	_, err := svc.CreateAdminUser(context.Background(), ctx, CreateAdminUserInput{
		Username:          "ops",
		TemporaryPassword: "temporary-password",
		Role:              RoleAdmin,
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeForbidden {
		t.Fatalf("create admin code = %s", appErr.Code)
	}

	result, err := svc.ListManagedUsers(context.Background(), ctx, ListManagedUsersInput{Role: RoleAdmin})
	if appErr := requireAppError(t, err); appErr.Code != CodeForbidden {
		t.Fatalf("list admin role code = %s, result = %+v", appErr.Code, result)
	}
}

func TestAdminCannotManageMultiRoleAdminRecord(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")
	target := UserRecord{
		User: User{
			ID:        "usr_ops",
			Username:  "ops",
			Status:    UserStatusActive,
			CreatedAt: repo.now,
			UpdatedAt: repo.now,
		},
		Roles:       []string{RoleStandard, RoleAdmin},
		Permissions: []string{PermissionSystemAdmin},
	}
	repo.usersByID[target.ID] = target
	repo.usersByUsername[target.Username] = target
	repo.credentials[target.ID] = mustChangeCredential(t, target.ID, "temporary-password")

	result, err := svc.ListManagedUsers(context.Background(), adminRequestContext(), ListManagedUsersInput{})
	if err != nil {
		t.Fatalf("ListManagedUsers() error = %v", err)
	}
	for _, user := range result.Users {
		if user.ID == target.ID {
			t.Fatalf("multi-role admin was listed for admin: %+v", user)
		}
	}

	_, err = svc.UpdateManagedUser(context.Background(), adminRequestContext(), target.ID, UpdateAdminUserInput{
		Status: OptionalStringField{Set: true, Value: stringPtr(UserStatusDisabled)},
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeNotFound {
		t.Fatalf("update code = %s", appErr.Code)
	}

	_, err = svc.ResetManagedUserPassword(context.Background(), adminRequestContext(), target.ID, ResetAdminPasswordInput{
		TemporaryPassword: "another-temporary",
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeNotFound {
		t.Fatalf("reset code = %s", appErr.Code)
	}
}

func TestResetManagedUserPasswordRecordsSecurityEvent(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")

	_, err := svc.ResetManagedUserPassword(context.Background(), adminRequestContext(), "usr_alice", ResetAdminPasswordInput{
		TemporaryPassword: "another-temporary",
	})
	if err != nil {
		t.Fatalf("ResetManagedUserPassword() error = %v", err)
	}
	if !repo.hasEvent(SecurityEventPasswordReset, SecurityEventStatusSuccess, reasonPasswordReset) {
		t.Fatalf("events = %+v", repo.events)
	}
}

func TestSuperAdminCanCreateAdminUser(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")
	ctx := testRequestContext()
	ctx.ActorUserID = "usr_super"
	ctx.ActorRoles = []string{RoleSuperAdmin}

	user, err := svc.CreateAdminUser(context.Background(), ctx, CreateAdminUserInput{
		Username:          "ops",
		TemporaryPassword: "temporary-password",
		Role:              RoleAdmin,
	})
	if err != nil {
		t.Fatalf("CreateAdminUser() error = %v", err)
	}
	if !userHasRole(user.UserRecord, RoleAdmin) {
		t.Fatalf("roles = %+v", user.Roles)
	}
}

func TestRequiredPasswordChangeVerifiesCurrentPasswordAndClearsFlag(t *testing.T) {
	repo := newFakeRepository(t)
	setMustChangePasswordCredential(t, repo, "usr_alice", "temporary-password")
	svc := newTestService(repo, "atk_v1_changed")

	_, err := svc.ChangeRequiredPassword(context.Background(), selfRequestContext("usr_alice"), "usr_alice", ChangePasswordInput{
		CurrentPassword:         "wrong-password",
		NewPassword:             "new-password",
		NewPasswordConfirmation: "new-password",
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeUnauthorized {
		t.Fatalf("code = %s", appErr.Code)
	}

	updated, err := svc.ChangeRequiredPassword(context.Background(), selfRequestContext("usr_alice"), "usr_alice", ChangePasswordInput{
		CurrentPassword:         "temporary-password",
		NewPassword:             "new-password",
		NewPasswordConfirmation: "new-password",
	})
	if err != nil {
		t.Fatalf("ChangeRequiredPassword() error = %v", err)
	}
	if updated.MustChangePassword {
		t.Fatalf("mustChangePassword = true")
	}
	if !repo.hasEvent(SecurityEventPasswordChanged, SecurityEventStatusSuccess, reasonPasswordChanged) {
		t.Fatalf("events = %+v", repo.events)
	}
}

func TestUpdateProfileRequiresGatewayCaller(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_created")
	ctx := selfRequestContext("usr_alice")
	ctx.CallerService = "knowledge"

	_, err := svc.UpdateProfile(context.Background(), ctx, "usr_alice", UpdateProfileInput{
		DisplayName: OptionalStringField{Set: true, Value: stringPtr("Alice Updated")},
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeForbidden {
		t.Fatalf("code = %s", appErr.Code)
	}
	if got := repo.usersByID["usr_alice"].DisplayName; got != "" {
		t.Fatalf("displayName changed to %q", got)
	}
}

func TestRequiredPasswordChangeRequiresGatewayCaller(t *testing.T) {
	repo := newFakeRepository(t)
	setMustChangePasswordCredential(t, repo, "usr_alice", "temporary-password")
	originalHash := repo.credentials["usr_alice"].PasswordHash
	svc := newTestService(repo, "atk_v1_changed")
	ctx := selfRequestContext("usr_alice")
	ctx.CallerService = "knowledge"

	_, err := svc.ChangeRequiredPassword(context.Background(), ctx, "usr_alice", ChangePasswordInput{
		CurrentPassword:         "temporary-password",
		NewPassword:             "new-password",
		NewPasswordConfirmation: "new-password",
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeForbidden {
		t.Fatalf("code = %s", appErr.Code)
	}
	if repo.credentials["usr_alice"].PasswordHash != originalHash {
		t.Fatalf("password hash changed")
	}
	if !repo.usersByID["usr_alice"].MustChangePassword {
		t.Fatalf("mustChangePassword cleared")
	}
}

func TestRequiredPasswordChangeRejectsUserWithoutRequiredChange(t *testing.T) {
	repo := newFakeRepository(t)
	originalHash := repo.credentials["usr_alice"].PasswordHash
	svc := newTestService(repo, "atk_v1_unchanged")

	_, err := svc.ChangeRequiredPassword(context.Background(), selfRequestContext("usr_alice"), "usr_alice", ChangePasswordInput{
		CurrentPassword:         "correct-password",
		NewPassword:             "new-password",
		NewPasswordConfirmation: "new-password",
	})
	if appErr := requireAppError(t, err); appErr.Code != CodeConflict {
		t.Fatalf("code = %s", appErr.Code)
	}
	if repo.credentials["usr_alice"].PasswordHash != originalHash {
		t.Fatalf("password hash changed")
	}
}

func TestRequiredPasswordChangeKeepsCurrentSessionActive(t *testing.T) {
	repo := newFakeRepository(t)
	setMustChangePasswordCredential(t, repo, "usr_alice", "temporary-password")
	svc := newTestService(repo, "atk_v1_existing")

	session, err := svc.CreateSession(context.Background(), testRequestContext(), CreateSessionInput{
		Username: "alice",
		Password: "temporary-password",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	updated, err := svc.ChangeRequiredPassword(context.Background(), selfRequestContext("usr_alice"), "usr_alice", ChangePasswordInput{
		CurrentPassword:         "temporary-password",
		NewPassword:             "new-password",
		NewPasswordConfirmation: "new-password",
	})
	if err != nil {
		t.Fatalf("ChangeRequiredPassword() error = %v", err)
	}
	if updated.MustChangePassword {
		t.Fatalf("mustChangePassword = true")
	}

	identity, err := svc.GetSessionByAccessToken(context.Background(), testRequestContext(), session.Session.AccessToken)
	if err != nil {
		t.Fatalf("GetSessionByAccessToken() after password change error = %v", err)
	}
	if identity.Session.ID != session.Session.SessionID || identity.User.MustChangePassword {
		t.Fatalf("identity = %+v", identity)
	}
}

func TestCreateUserReturnsSuccessWhenSecurityEventWriteFails(t *testing.T) {
	repo := newFakeRepository(t)
	repo.eventErr = errors.New("security event store unavailable")
	svc := newTestService(repo, "atk_v1_created")

	result, err := svc.CreateUser(context.Background(), testRequestContext(), CreateUserInput{
		Username: "bob",
		Password: "bob-password",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if result.User.ID == "" || result.Session.SessionID == "" || result.Session.AccessToken != "atk_v1_created" {
		t.Fatalf("result = %+v", result)
	}
	if _, ok := repo.usersByUsername["bob"]; !ok {
		t.Fatalf("user was not persisted")
	}
	if _, ok := repo.sessions[result.Session.SessionID]; !ok {
		t.Fatalf("session was not persisted")
	}
}

func TestCreateSessionReturnsSuccessWhenSecurityEventWriteFails(t *testing.T) {
	repo := newFakeRepository(t)
	repo.eventErr = errors.New("security event store unavailable")
	svc := newTestService(repo, "atk_v1_created")

	result, err := svc.CreateSession(context.Background(), testRequestContext(), CreateSessionInput{
		Username: "alice",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if result.Session.SessionID == "" || result.Session.AccessToken != "atk_v1_created" {
		t.Fatalf("session = %+v", result.Session)
	}
	if _, ok := repo.sessions[result.Session.SessionID]; !ok {
		t.Fatalf("session was not persisted")
	}
}

func TestRevokedTokenNoLongerReturnsActiveSession(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_revoked")

	result, err := svc.CreateSession(context.Background(), testRequestContext(), CreateSessionInput{
		Username: "alice",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, err := svc.GetSessionByAccessToken(context.Background(), testRequestContext(), result.Session.AccessToken); err != nil {
		t.Fatalf("GetSessionByAccessToken() before revoke error = %v", err)
	}
	if err := svc.RevokeSession(context.Background(), testRequestContext(), result.Session.SessionID, "user_logout"); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	_, err = svc.GetSessionByAccessToken(context.Background(), testRequestContext(), result.Session.AccessToken)
	if appErr := requireAppError(t, err); appErr.Code != CodeUnauthorized {
		t.Fatalf("code = %s", appErr.Code)
	}
	if !repo.hasEvent(SecurityEventSessionRevoked, SecurityEventStatusSuccess, "user_logout") {
		t.Fatalf("events = %+v", repo.events)
	}
}

func TestGetSessionRejectsRevokedSessionByID(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_revoked")

	result, err := svc.CreateSession(context.Background(), testRequestContext(), CreateSessionInput{
		Username: "alice",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := svc.RevokeSession(context.Background(), testRequestContext(), result.Session.SessionID, "user_logout"); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}

	_, err = svc.GetSession(context.Background(), testRequestContext(), result.Session.SessionID)
	if appErr := requireAppError(t, err); appErr.Code != CodeNotFound {
		t.Fatalf("code = %s", appErr.Code)
	}
}

func TestRevokeSessionReturnsSuccessWhenSecurityEventWriteFails(t *testing.T) {
	repo := newFakeRepository(t)
	svc := newTestService(repo, "atk_v1_revoked")

	result, err := svc.CreateSession(context.Background(), testRequestContext(), CreateSessionInput{
		Username: "alice",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	repo.eventErr = errors.New("security event store unavailable")

	if err := svc.RevokeSession(context.Background(), testRequestContext(), result.Session.SessionID, "user_logout"); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	if got := repo.sessions[result.Session.SessionID].Status; got != SessionStatusRevoked {
		t.Fatalf("session status = %q", got)
	}
}

func newTestService(repo *fakeRepository, token string) *Service {
	now := time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)
	counter := map[string]int{}
	return New(repo,
		WithClock(func() time.Time { return now }),
		WithTokenGenerator(func() (string, error) { return token, nil }),
		WithTokenHashSecret([]byte("test-token-hash-secret")),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		WithIDGenerator(func(prefix string) string {
			counter[prefix]++
			return prefix + "_" + strconv.Itoa(counter[prefix])
		}),
	)
}

func newFakeRepository(t *testing.T) *fakeRepository {
	t.Helper()
	now := time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)
	hash, err := hashPassword("correct-password")
	if err != nil {
		t.Fatalf("hashPassword() error = %v", err)
	}
	user := UserRecord{
		User: User{
			ID:        "usr_alice",
			Username:  "alice",
			Status:    UserStatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Roles:       []string{"standard"},
		Permissions: []string{"knowledge:read"},
	}
	admin := adminUserRecord("usr_admin", "admin", RoleAdmin, now)
	superAdmin := adminUserRecord("usr_super", "super", RoleSuperAdmin, now)
	return &fakeRepository{
		now:             now,
		usersByID:       map[string]UserRecord{user.ID: user, admin.ID: admin, superAdmin.ID: superAdmin},
		usersByUsername: map[string]UserRecord{user.Username: user, admin.Username: admin, superAdmin.Username: superAdmin},
		credentials: map[string]Credential{
			user.ID: {
				ID:                        "cred_alice",
				UserID:                    user.ID,
				CredentialType:            CredentialTypePassword,
				PasswordHash:              hash,
				PasswordHashAlg:           PasswordHashAlg,
				PasswordHashParamsVersion: PasswordHashParamsVersion,
			},
		},
		sessions:     map[string]Session{},
		activeByHash: map[string]string{},
	}
}

func adminUserRecord(id string, username string, role string, now time.Time) UserRecord {
	return UserRecord{
		User: User{
			ID:        id,
			Username:  username,
			Status:    UserStatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Roles:       []string{role},
		Permissions: []string{PermissionSystemAdmin},
	}
}

func testRequestContext() RequestContext {
	return RequestContext{
		RequestID:     "req_test",
		CallerService: "gateway",
		ClientIP:      "127.0.0.1",
		UserAgent:     "auth-test",
	}
}

func adminRequestContext() RequestContext {
	ctx := testRequestContext()
	ctx.ActorUserID = "usr_admin"
	ctx.ActorRoles = []string{RoleAdmin}
	return ctx
}

func selfRequestContext(userID string) RequestContext {
	ctx := testRequestContext()
	ctx.ActorUserID = userID
	return ctx
}

func mustChangeCredential(t *testing.T, userID string, password string) Credential {
	t.Helper()
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword() error = %v", err)
	}
	return Credential{
		ID:                        "cred_" + userID,
		UserID:                    userID,
		CredentialType:            CredentialTypePassword,
		PasswordHash:              hash,
		PasswordHashAlg:           PasswordHashAlg,
		PasswordHashParamsVersion: PasswordHashParamsVersion,
		MustChangePassword:        true,
	}
}

func setMustChangePasswordCredential(t *testing.T, repo *fakeRepository, userID string, password string) {
	t.Helper()
	repo.credentials[userID] = mustChangeCredential(t, userID, password)
	user, ok := repo.usersByID[userID]
	if !ok {
		t.Fatalf("user %q not found", userID)
	}
	user.MustChangePassword = true
	repo.usersByID[userID] = user
	repo.usersByUsername[user.Username] = user
}

func requireAppError(t *testing.T, err error) *AppError {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error is not AppError: %T %v", err, err)
	}
	return appErr
}

type fakeRepository struct {
	now             time.Time
	usersByID       map[string]UserRecord
	usersByUsername map[string]UserRecord
	credentials     map[string]Credential
	sessions        map[string]Session
	activeByHash    map[string]string
	events          []SecurityEventParams
	eventErr        error
}

func (r *fakeRepository) FindUserByID(_ context.Context, id string) (UserRecord, error) {
	user, ok := r.usersByID[id]
	if !ok {
		return UserRecord{}, ErrNotFound
	}
	return user, nil
}

func (r *fakeRepository) FindUserByUsername(_ context.Context, username string) (UserRecord, error) {
	user, ok := r.usersByUsername[username]
	if !ok {
		return UserRecord{}, ErrNotFound
	}
	return user, nil
}

func (r *fakeRepository) FindCredentialByUserID(_ context.Context, userID string) (Credential, error) {
	credential, ok := r.credentials[userID]
	if !ok {
		return Credential{}, ErrNotFound
	}
	return credential, nil
}

func (r *fakeRepository) FindSessionByID(_ context.Context, id string) (SessionIdentity, error) {
	session, ok := r.sessions[id]
	if !ok {
		return SessionIdentity{}, ErrNotFound
	}
	user, err := r.FindUserByID(context.Background(), session.UserID)
	if err != nil {
		return SessionIdentity{}, err
	}
	return SessionIdentity{Session: session, User: summaryFromRecord(user), AccessTokenHash: session.AccessTokenHash}, nil
}

func (r *fakeRepository) FindActiveSessionByTokenHash(ctx context.Context, tokenHash string) (SessionIdentity, error) {
	sessionID, ok := r.activeByHash[tokenHash]
	if !ok {
		return SessionIdentity{}, ErrNotFound
	}
	session, ok := r.sessions[sessionID]
	if !ok || session.Status != SessionStatusActive || !session.ExpiresAt.After(r.now) {
		return SessionIdentity{}, ErrNotFound
	}
	return r.FindSessionByID(ctx, session.ID)
}

func (r *fakeRepository) ListManagedUsers(_ context.Context, params ListManagedUsersParams) ([]UserRecord, int64, error) {
	users := []UserRecord{}
	for _, user := range r.usersByID {
		if user.ID == params.ActorUserID {
			continue
		}
		hasUnmanageableRole := false
		for _, role := range params.ManagedRoles {
			if hasStringFold(user.Roles, role) && !hasStringFold(params.ManageableRoles, role) {
				hasUnmanageableRole = true
				break
			}
		}
		if hasUnmanageableRole {
			continue
		}
		match := false
		for _, role := range params.ManageableRoles {
			if hasStringFold(user.Roles, role) && (params.Role == "" || hasStringFold(user.Roles, params.Role)) {
				match = true
			}
		}
		if !match {
			continue
		}
		users = append(users, user)
	}
	return users, int64(len(users)), nil
}

func (r *fakeRepository) CreateUserWithCredential(_ context.Context, params CreateUserParams) (UserRecord, error) {
	if _, exists := r.usersByUsername[params.Username]; exists {
		return UserRecord{}, ErrConflict
	}
	user := UserRecord{
		User: User{
			ID:          params.ID,
			Username:    params.Username,
			DisplayName: params.DisplayName,
			Status:      params.Status,
			CreatedAt:   params.CreatedAt,
			UpdatedAt:   params.CreatedAt,
		},
		MustChangePassword: params.MustChangePassword,
		Roles:              []string{params.DefaultRoleCode},
		Permissions:        []string{"knowledge:read"},
	}
	r.usersByID[user.ID] = user
	r.usersByUsername[user.Username] = user
	r.credentials[user.ID] = Credential{
		ID:                        params.PasswordCredentialID,
		UserID:                    user.ID,
		CredentialType:            CredentialTypePassword,
		PasswordHash:              params.PasswordHash,
		PasswordHashAlg:           params.PasswordHashAlg,
		PasswordHashParamsVersion: params.PasswordHashParamsVersion,
		MustChangePassword:        params.MustChangePassword,
	}
	return user, nil
}

func (r *fakeRepository) UpdateUserProfile(_ context.Context, params UpdateUserProfileParams) (UserRecord, error) {
	user, ok := r.usersByID[params.UserID]
	if !ok {
		return UserRecord{}, ErrNotFound
	}
	user.DisplayName = params.DisplayName
	user.Email = params.Email
	user.Phone = params.Phone
	user.UpdatedAt = params.UpdatedAt
	r.usersByID[user.ID] = user
	r.usersByUsername[user.Username] = user
	return user, nil
}

func (r *fakeRepository) UpdateUserStatus(_ context.Context, params UpdateUserStatusParams) (UserRecord, error) {
	user, ok := r.usersByID[params.UserID]
	if !ok {
		return UserRecord{}, ErrNotFound
	}
	user.Status = params.Status
	user.UpdatedAt = params.UpdatedAt
	r.usersByID[user.ID] = user
	r.usersByUsername[user.Username] = user
	return user, nil
}

func (r *fakeRepository) ReplaceUserRole(_ context.Context, params ReplaceUserRoleParams) (UserRecord, error) {
	user, ok := r.usersByID[params.UserID]
	if !ok {
		return UserRecord{}, ErrNotFound
	}
	roles := []string{}
	for _, role := range user.Roles {
		if !hasStringFold(params.ManagedRoleCodes, role) {
			roles = append(roles, role)
		}
	}
	roles = append(roles, params.RoleCode)
	user.Roles = roles
	r.usersByID[user.ID] = user
	r.usersByUsername[user.Username] = user
	return user, nil
}

func (r *fakeRepository) UpdatePassword(_ context.Context, params UpdatePasswordParams) (Credential, error) {
	credential, ok := r.credentials[params.UserID]
	if !ok {
		return Credential{}, ErrNotFound
	}
	credential.PasswordHash = params.PasswordHash
	credential.PasswordHashAlg = params.PasswordHashAlg
	credential.PasswordHashParamsVersion = params.PasswordHashParamsVersion
	credential.PasswordHashParamsJSON = params.PasswordHashParamsJSON
	credential.MustChangePassword = params.MustChangePassword
	credential.PasswordChangedAt = params.ChangedAt
	credential.UpdatedAt = params.ChangedAt
	r.credentials[params.UserID] = credential
	user := r.usersByID[params.UserID]
	user.MustChangePassword = params.MustChangePassword
	r.usersByID[params.UserID] = user
	r.usersByUsername[user.Username] = user
	return credential, nil
}

func (r *fakeRepository) CreateSession(_ context.Context, params CreateSessionParams) (SessionIdentity, error) {
	if _, ok := r.usersByID[params.UserID]; !ok {
		return SessionIdentity{}, ErrNotFound
	}
	session := Session{
		ID:                        params.ID,
		UserID:                    params.UserID,
		AccessTokenHash:           params.AccessTokenHash,
		AccessTokenHashAlg:        params.AccessTokenHashAlg,
		AccessTokenHashKeyVersion: params.AccessTokenHashKeyVersion,
		TokenType:                 TokenTypeBearer,
		Status:                    SessionStatusActive,
		IssuedAt:                  params.IssuedAt,
		ExpiresAt:                 params.ExpiresAt,
		ClientIP:                  params.ClientIP,
		UserAgent:                 params.UserAgent,
		CreatedRequestID:          params.RequestID,
		CreatedAt:                 params.IssuedAt,
		UpdatedAt:                 params.IssuedAt,
	}
	r.sessions[session.ID] = session
	r.activeByHash[session.AccessTokenHash] = session.ID
	return r.FindSessionByID(context.Background(), session.ID)
}

func (r *fakeRepository) RevokeSession(_ context.Context, params RevokeSessionParams) (Session, error) {
	session, ok := r.sessions[params.SessionID]
	if !ok || session.Status != SessionStatusActive {
		return Session{}, ErrNotFound
	}
	session.Status = SessionStatusRevoked
	session.RevokedAt = &params.RevokedAt
	session.RevokeReason = &params.Reason
	session.RevokedRequestID = params.RequestID
	session.UpdatedAt = params.RevokedAt
	r.sessions[session.ID] = session
	delete(r.activeByHash, session.AccessTokenHash)
	return session, nil
}

func (r *fakeRepository) RevokeUserSessions(_ context.Context, params RevokeUserSessionsParams) ([]Session, error) {
	revoked := []Session{}
	for id, session := range r.sessions {
		if session.UserID != params.UserID || session.Status != SessionStatusActive {
			continue
		}
		session.Status = SessionStatusRevoked
		session.RevokedAt = &params.RevokedAt
		session.RevokeReason = &params.Reason
		session.RevokedRequestID = params.RequestID
		session.UpdatedAt = params.RevokedAt
		r.sessions[id] = session
		delete(r.activeByHash, session.AccessTokenHash)
		revoked = append(revoked, session)
	}
	return revoked, nil
}

func (r *fakeRepository) RecordSecurityEvent(_ context.Context, params SecurityEventParams) error {
	r.events = append(r.events, params)
	if r.eventErr != nil {
		return r.eventErr
	}
	return nil
}

func (r *fakeRepository) hasEvent(eventType string, status string, reason string) bool {
	for _, event := range r.events {
		if event.EventType != eventType || event.Status != status {
			continue
		}
		if reason == "" {
			return true
		}
		if event.ReasonCode != nil && *event.ReasonCode == reason {
			return true
		}
	}
	return false
}
