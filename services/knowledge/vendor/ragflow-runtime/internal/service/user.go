//
//  Copyright 2026 The InfiniFlow Authors. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

package service

import (
	"context"
	"fmt"
	"ragflow/internal/common"
	"ragflow/internal/dao"
	"ragflow/internal/engine/redis"
	"ragflow/internal/entity"
	"ragflow/internal/server"
	"ragflow/internal/utility"
	"strings"
)

// UserService user service
type UserService struct {
	userDAO *dao.UserDAO
}

// NewUserService create user service
func NewUserService() *UserService {
	return &UserService{
		userDAO: dao.NewUserDAO(),
	}
}

// GetUserByToken gets user by authorization header
// The token parameter is the authorization header value, which needs to be decrypted
// using itsdangerous URLSafeTimedSerializer to get the actual access_token
func (s *UserService) GetUserByToken(authorization string) (*entity.User, common.ErrorCode, error) {
	secretKey, err := server.GetSecretKey(redis.Get())
	if err != nil {
		return nil, common.CodeUnauthorized, err
	}

	accessToken, err := utility.ExtractAccessToken(authorization, secretKey)
	if err != nil {
		return nil, common.CodeUnauthorized, fmt.Errorf("invalid authorization token: %w", err)
	}

	if len(accessToken) < 32 {
		return nil, common.CodeUnauthorized, fmt.Errorf("invalid access token format")
	}

	user, err := s.userDAO.GetByAccessToken(accessToken)
	if err != nil {
		return nil, common.CodeUnauthorized, err
	}

	return user, common.CodeSuccess, nil
}

// GetUserByAPIToken gets user by access key from Authorization header
// This is used for API token authentication
// The authorization parameter should be in format: "Bearer <token>" or just "<token>"
func (s *UserService) GetUserByAPIToken(authorization string) (*entity.User, common.ErrorCode, error) {
	if authorization == "" {
		return nil, common.CodeUnauthorized, fmt.Errorf("authorization header is empty")
	}

	parts := strings.Split(authorization, " ")
	var token string
	if len(parts) == 2 {
		token = parts[1]
	} else if len(parts) == 1 {
		token = parts[0]
	} else {
		return nil, common.CodeUnauthorized, fmt.Errorf("invalid authorization format")
	}

	apiTokenDAO := dao.NewAPITokenDAO()
	userToken, err := apiTokenDAO.GetUserByAPIToken(token)
	if err != nil {
		return nil, common.CodeUnauthorized, fmt.Errorf("invalid access token")
	}

	user, err := s.userDAO.GetByTenantID(userToken.TenantID)
	if err != nil {
		return nil, common.CodeUnauthorized, fmt.Errorf("user not found for this access token")
	}

	if user.AccessToken == nil || *user.AccessToken == "" {
		return nil, common.CodeUnauthorized, fmt.Errorf("user has empty access_token in database")
	}

	return user, common.CodeSuccess, nil
}

// GetUserByBetaAPIToken gets user by beta access key from Authorization
// header. This mirrors Python's AUTH_BETA flow used by beta-token endpoints.
func (s *UserService) GetUserByBetaAPIToken(authorization string) (*entity.User, common.ErrorCode, error) {
	authorization = strings.TrimSpace(authorization)
	if authorization == "" {
		return nil, common.CodeUnauthorized, fmt.Errorf("authorization header is empty")
	}

	parts := strings.Fields(authorization)
	var token string
	if len(parts) == 2 {
		token = parts[1]
	} else if len(parts) == 1 {
		if strings.EqualFold(parts[0], "Bearer") {
			return nil, common.CodeUnauthorized, fmt.Errorf("invalid authorization format")
		}
		token = parts[0]
	} else {
		return nil, common.CodeUnauthorized, fmt.Errorf("invalid authorization format")
	}
	if token == "" {
		return nil, common.CodeUnauthorized, fmt.Errorf("invalid authorization format")
	}

	apiTokenDAO := dao.NewAPITokenDAO()
	userToken, err := apiTokenDAO.GetByBeta(token)
	if err != nil {
		return nil, common.CodeUnauthorized, fmt.Errorf("invalid beta access token")
	}

	user, err := s.userDAO.GetByTenantID(userToken.TenantID)
	if err != nil {
		return nil, common.CodeUnauthorized, fmt.Errorf("user not found for this beta access token")
	}

	if user.AccessToken == nil || *user.AccessToken == "" {
		return nil, common.CodeUnauthorized, fmt.Errorf("user has empty access_token in database")
	}

	return user, common.CodeSuccess, nil
}

// UserTenantService user tenant service
// Provides business logic for user-tenant relationship management
type UserTenantService struct {
	userTenantDAO *dao.UserTenantDAO
}

// NewUserTenantService creates a new UserTenantService instance
func NewUserTenantService() *UserTenantService {
	return &UserTenantService{
		userTenantDAO: dao.NewUserTenantDAO(),
	}
}

// UserTenantRelation represents a user-tenant relationship response
// This structure matches the Python implementation's return format
type UserTenantRelation struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
}

// GetUserTenantRelationByUserID retrieves all user-tenant relationships for a given user ID
func (s *UserTenantService) GetUserTenantRelationByUserID(userID string) ([]*UserTenantRelation, error) {
	return s.GetUserTenantRelationByUserIDWithContext(context.Background(), userID)
}

// GetUserTenantRelationByUserIDWithContext retrieves all user-tenant relationships for a given user ID with context.
func (s *UserTenantService) GetUserTenantRelationByUserIDWithContext(ctx context.Context, userID string) ([]*UserTenantRelation, error) {
	relations, err := s.userTenantDAO.GetByUserIDWithContext(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*UserTenantRelation, len(relations))
	for i, rel := range relations {
		result[i] = convertToUserTenantRelation(rel)
	}

	return result, nil
}

func convertToUserTenantRelation(userTenant *entity.UserTenant) *UserTenantRelation {
	return &UserTenantRelation{
		ID:       userTenant.ID,
		UserID:   userTenant.UserID,
		TenantID: userTenant.TenantID,
		Role:     userTenant.Role,
	}
}
