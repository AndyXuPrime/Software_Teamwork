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
	"strings"

	"ragflow/internal/common"
	"ragflow/internal/dao"
	"ragflow/internal/entity"
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

// GetUserByTenantID resolves the runtime user row for a gateway-injected tenant id.
func (s *UserService) GetUserByTenantID(tenantID string) (*entity.User, common.ErrorCode, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, common.CodeUnauthorized, fmt.Errorf("tenant id is empty")
	}

	user, err := s.userDAO.GetByTenantID(tenantID)
	if err != nil {
		return nil, common.CodeUnauthorized, err
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
