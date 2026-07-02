-- name: GetUserByID :one
SELECT
    id,
    username,
    display_name,
    email,
    phone,
    status,
    locked_until,
    last_login_at,
    created_at,
    updated_at,
    deleted_at
FROM auth_users
WHERE id = $1
    AND deleted_at IS NULL;

-- name: GetUserByUsername :one
SELECT
    id,
    username,
    display_name,
    email,
    phone,
    status,
    locked_until,
    last_login_at,
    created_at,
    updated_at,
    deleted_at
FROM auth_users
WHERE username = $1
    AND deleted_at IS NULL;

-- name: GetCredentialByUserID :one
SELECT
    id,
    user_id,
    credential_type,
    password_hash,
    password_hash_alg,
    password_hash_params_version,
    password_hash_params_json,
    must_change_password,
    password_changed_at,
    password_expires_at,
    failed_attempt_count,
    last_failed_at,
    created_at,
    updated_at
FROM auth_credentials
WHERE user_id = $1
    AND credential_type = $2;

-- name: ListRoleCodesByUserID :many
SELECT
    r.code
FROM user_roles ur
INNER JOIN auth_roles r
    ON r.id = ur.role_id
WHERE ur.user_id = $1
    AND r.enabled = TRUE
    AND (ur.expires_at IS NULL OR ur.expires_at > now())
ORDER BY r.code ASC;

-- name: ListPermissionCodesByUserID :many
SELECT DISTINCT
    p.code
FROM user_roles ur
INNER JOIN auth_roles r
    ON r.id = ur.role_id
INNER JOIN role_permissions rp
    ON rp.role_id = r.id
INNER JOIN auth_permissions p
    ON p.id = rp.permission_id
WHERE ur.user_id = $1
    AND r.enabled = TRUE
    AND p.enabled = TRUE
    AND (ur.expires_at IS NULL OR ur.expires_at > now())
ORDER BY p.code ASC;

-- name: CountManagedUsers :one
SELECT count(*)::bigint
FROM auth_users u
WHERE u.deleted_at IS NULL
    AND u.id <> @actor_user_id
    AND (@username::text = '' OR u.username ILIKE '%' || @username::text || '%')
    AND (@status::text = '' OR u.status = @status::text)
    AND NOT EXISTS (
        SELECT 1
        FROM user_roles ur
        INNER JOIN auth_roles r ON r.id = ur.role_id
        WHERE ur.user_id = u.id
            AND r.code = ANY(@managed_roles::text[])
            AND NOT (r.code = ANY(@manageable_roles::text[]))
            AND r.enabled = TRUE
            AND (ur.expires_at IS NULL OR ur.expires_at > now())
    )
    AND EXISTS (
        SELECT 1
        FROM user_roles ur
        INNER JOIN auth_roles r ON r.id = ur.role_id
        WHERE ur.user_id = u.id
            AND r.code = ANY(@manageable_roles::text[])
            AND (@role::text = '' OR r.code = @role::text)
            AND r.enabled = TRUE
            AND (ur.expires_at IS NULL OR ur.expires_at > now())
    );

-- name: ListManagedUsers :many
SELECT
    u.id,
    u.username,
    u.display_name,
    u.email,
    u.phone,
    u.status,
    u.locked_until,
    u.last_login_at,
    u.created_at,
    u.updated_at,
    u.deleted_at
FROM auth_users u
WHERE u.deleted_at IS NULL
    AND u.id <> @actor_user_id
    AND (@username::text = '' OR u.username ILIKE '%' || @username::text || '%')
    AND (@status::text = '' OR u.status = @status::text)
    AND NOT EXISTS (
        SELECT 1
        FROM user_roles ur
        INNER JOIN auth_roles r ON r.id = ur.role_id
        WHERE ur.user_id = u.id
            AND r.code = ANY(@managed_roles::text[])
            AND NOT (r.code = ANY(@manageable_roles::text[]))
            AND r.enabled = TRUE
            AND (ur.expires_at IS NULL OR ur.expires_at > now())
    )
    AND EXISTS (
        SELECT 1
        FROM user_roles ur
        INNER JOIN auth_roles r ON r.id = ur.role_id
        WHERE ur.user_id = u.id
            AND r.code = ANY(@manageable_roles::text[])
            AND (@role::text = '' OR r.code = @role::text)
            AND r.enabled = TRUE
            AND (ur.expires_at IS NULL OR ur.expires_at > now())
    )
ORDER BY u.created_at DESC, u.id ASC
LIMIT @limit_count
OFFSET @offset_count;

-- name: GetSessionByID :one
SELECT
    s.id,
    s.user_id,
    s.access_token_hash,
    s.access_token_hash_alg,
    s.access_token_hash_key_version,
    s.token_type,
    s.status,
    s.issued_at,
    s.expires_at,
    s.last_seen_at,
    s.revoked_at,
    s.revoke_reason,
    s.client_ip,
    s.user_agent,
    s.created_request_id,
    s.revoked_request_id,
    s.created_at,
    s.updated_at
FROM auth_sessions s
WHERE s.id = $1;

-- name: GetActiveSessionByTokenHash :one
SELECT
    s.id,
    s.user_id,
    s.access_token_hash,
    s.access_token_hash_alg,
    s.access_token_hash_key_version,
    s.token_type,
    s.status,
    s.issued_at,
    s.expires_at,
    s.last_seen_at,
    s.revoked_at,
    s.revoke_reason,
    s.client_ip,
    s.user_agent,
    s.created_request_id,
    s.revoked_request_id,
    s.created_at,
    s.updated_at
FROM auth_sessions s
WHERE s.access_token_hash = $1
    AND s.status = 'active'
    AND s.expires_at > now();

-- name: CreateUser :one
INSERT INTO auth_users (
    id,
    username,
    display_name,
    email,
    phone,
    status,
    locked_until,
    last_login_at,
    created_at,
    updated_at,
    deleted_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULL)
RETURNING
    id,
    username,
    display_name,
    email,
    phone,
    status,
    locked_until,
    last_login_at,
    created_at,
    updated_at,
    deleted_at;

-- name: CreateCredential :one
INSERT INTO auth_credentials (
    id,
    user_id,
    credential_type,
    password_hash,
    password_hash_alg,
    password_hash_params_version,
    password_hash_params_json,
    must_change_password,
    password_changed_at,
    password_expires_at,
    failed_attempt_count,
    last_failed_at,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, 0, NULL, $11, $12)
RETURNING
    id,
    user_id,
    credential_type,
    password_hash,
    password_hash_alg,
    password_hash_params_version,
    password_hash_params_json,
    must_change_password,
    password_changed_at,
    password_expires_at,
    failed_attempt_count,
    last_failed_at,
    created_at,
    updated_at;

-- name: AssignRoleByCode :one
INSERT INTO user_roles (
    id,
    user_id,
    role_id,
    assigned_by,
    assigned_at,
    expires_at,
    created_at
)
SELECT
    $1,
    $2,
    r.id,
    $4,
    $5,
    NULL,
    $6
FROM auth_roles r
WHERE r.code = $3
    AND r.enabled = TRUE
ON CONFLICT (user_id, role_id) DO UPDATE
SET assigned_by = EXCLUDED.assigned_by
RETURNING
    id,
    user_id,
    role_id,
    assigned_by,
    assigned_at,
    expires_at,
    created_at;

-- name: CreateSession :one
INSERT INTO auth_sessions (
    id,
    user_id,
    access_token_hash,
    access_token_hash_alg,
    access_token_hash_key_version,
    token_type,
    status,
    issued_at,
    expires_at,
    last_seen_at,
    revoked_at,
    revoke_reason,
    client_ip,
    user_agent,
    created_request_id,
    revoked_request_id,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, 'Bearer', 'active', $6, $7, NULL, NULL, NULL, $8, $9, $10, NULL, $11, $12)
RETURNING
    id,
    user_id,
    access_token_hash,
    access_token_hash_alg,
    access_token_hash_key_version,
    token_type,
    status,
    issued_at,
    expires_at,
    last_seen_at,
    revoked_at,
    revoke_reason,
    client_ip,
    user_agent,
    created_request_id,
    revoked_request_id,
    created_at,
    updated_at;

-- name: UpdateUserProfile :one
UPDATE auth_users
SET display_name = $2,
    email = $3,
    phone = $4,
    updated_at = $5
WHERE id = $1
    AND deleted_at IS NULL
RETURNING
    id,
    username,
    display_name,
    email,
    phone,
    status,
    locked_until,
    last_login_at,
    created_at,
    updated_at,
    deleted_at;

-- name: UpdateUserStatus :one
UPDATE auth_users
SET status = $2,
    updated_at = $3
WHERE id = $1
    AND deleted_at IS NULL
RETURNING
    id,
    username,
    display_name,
    email,
    phone,
    status,
    locked_until,
    last_login_at,
    created_at,
    updated_at,
    deleted_at;

-- name: DeleteManagedUserRoles :exec
DELETE FROM user_roles
WHERE user_id = $1
    AND role_id IN (
        SELECT id
        FROM auth_roles
        WHERE code = ANY(@role_codes::text[])
    );

-- name: UpdateCredentialPassword :one
UPDATE auth_credentials
SET password_hash = $3,
    password_hash_alg = $4,
    password_hash_params_version = $5,
    password_hash_params_json = $6::jsonb,
    must_change_password = $7,
    password_changed_at = $8,
    failed_attempt_count = 0,
    last_failed_at = NULL,
    updated_at = $8
WHERE user_id = $1
    AND credential_type = $2
RETURNING
    id,
    user_id,
    credential_type,
    password_hash,
    password_hash_alg,
    password_hash_params_version,
    password_hash_params_json,
    must_change_password,
    password_changed_at,
    password_expires_at,
    failed_attempt_count,
    last_failed_at,
    created_at,
    updated_at;

-- name: UpdateUserLastLoginAt :exec
UPDATE auth_users
SET last_login_at = $2,
    updated_at = $2
WHERE id = $1
    AND deleted_at IS NULL;

-- name: RevokeSession :one
UPDATE auth_sessions
SET status = 'revoked',
    revoked_at = $2,
    revoke_reason = $3,
    revoked_request_id = $4,
    updated_at = $2
WHERE id = $1
    AND status = 'active'
RETURNING
    id,
    user_id,
    access_token_hash,
    access_token_hash_alg,
    access_token_hash_key_version,
    token_type,
    status,
    issued_at,
    expires_at,
    last_seen_at,
    revoked_at,
    revoke_reason,
    client_ip,
    user_agent,
    created_request_id,
    revoked_request_id,
    created_at,
    updated_at;

-- name: RevokeActiveSessionsForUser :many
UPDATE auth_sessions
SET status = 'revoked',
    revoked_at = $2,
    revoke_reason = $3,
    revoked_request_id = $4,
    updated_at = $2
WHERE user_id = $1
    AND status = 'active'
RETURNING
    id,
    user_id,
    access_token_hash,
    access_token_hash_alg,
    access_token_hash_key_version,
    token_type,
    status,
    issued_at,
    expires_at,
    last_seen_at,
    revoked_at,
    revoke_reason,
    client_ip,
    user_agent,
    created_request_id,
    revoked_request_id,
    created_at,
    updated_at;

-- name: CreateSessionRevocation :exec
INSERT INTO session_revocations (
    id,
    session_id,
    user_id,
    reason,
    revoked_by,
    request_id,
    revoked_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (session_id) DO NOTHING;

-- name: CreateSecurityEvent :exec
INSERT INTO auth_security_events (
    id,
    event_type,
    user_id,
    session_id,
    username_snapshot,
    request_id,
    client_ip,
    user_agent,
    caller_service,
    status,
    reason_code,
    metadata_json,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb, $13);
