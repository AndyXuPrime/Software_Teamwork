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

package handler

import (
	"net/http"
	"strings"

	"ragflow/internal/common"
	"ragflow/internal/entity"
	"ragflow/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	gatewayTenantHeader = "X-Tenant-Id"
	gatewayUserHeader   = "X-User-Id"
)

// AuthHandler resolves tenant context injected by an upstream gateway.
type AuthHandler struct {
	userService tenantUserResolver
}

type tenantUserResolver interface {
	GetUserByTenantID(tenantID string) (*entity.User, common.ErrorCode, error)
}

// NewAuthHandler create auth handler
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		userService: service.NewUserService(),
	}
}

func gatewayTenantID(c *gin.Context) string {
	if tenantID := strings.TrimSpace(c.GetHeader(gatewayTenantHeader)); tenantID != "" {
		return tenantID
	}
	return strings.TrimSpace(c.GetHeader(gatewayUserHeader))
}

func (h *AuthHandler) resolveGatewayUser(c *gin.Context) (*entity.User, bool) {
	tenantID := gatewayTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    common.CodeUnauthorized,
			"message": "Missing X-Tenant-Id header",
		})
		return nil, false
	}

	user, code, err := h.userService.GetUserByTenantID(tenantID)
	if err != nil || code != common.CodeSuccess || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    common.CodeUnauthorized,
			"message": "Tenant not found",
		})
		return nil, false
	}

	if user.IsSuperuser != nil && *user.IsSuperuser {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    common.CodeForbidden,
			"message": "Super user shouldn't access the URL",
		})
		return nil, false
	}

	c.Set("user", user)
	c.Set("user_id", user.ID)
	c.Set("email", user.Email)
	return user, true
}

// BetaAuthMiddleware uses the same gateway tenant header contract as AuthMiddleware.
func (h *AuthHandler) BetaAuthMiddleware() gin.HandlerFunc {
	return h.AuthMiddleware()
}

// AuthMiddleware trusts tenant identity from the upstream gateway.
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := h.resolveGatewayUser(c); !ok {
			c.Abort()
			return
		}
		c.Next()
	}
}
