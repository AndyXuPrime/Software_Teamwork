-- +goose Up
INSERT INTO auth_permissions (id, code, domain, action, description, enabled, created_at, updated_at)
VALUES
    ('perm_qa_settings_read', 'qa:settings:read', 'qa', 'settings:read', 'Read QA runtime settings.', TRUE, now(), now()),
    ('perm_qa_settings_write', 'qa:settings:write', 'qa', 'settings:write', 'Manage QA runtime settings.', TRUE, now(), now())
ON CONFLICT (code) DO UPDATE
SET domain = EXCLUDED.domain,
    action = EXCLUDED.action,
    description = EXCLUDED.description,
    enabled = EXCLUDED.enabled,
    updated_at = now();

INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT seed.id, r.id, p.id, now()
FROM (
    VALUES
        ('rperm_admin_qa_settings_read', 'admin', 'qa:settings:read'),
        ('rperm_admin_qa_settings_write', 'admin', 'qa:settings:write'),
        ('rperm_super_qa_settings_read', 'super_admin', 'qa:settings:read'),
        ('rperm_super_qa_settings_write', 'super_admin', 'qa:settings:write')
) AS seed(id, role_code, permission_code)
INNER JOIN auth_roles r ON r.code = seed.role_code
INNER JOIN auth_permissions p ON p.code = seed.permission_code
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE role_id IN (
    SELECT id FROM auth_roles WHERE code IN ('admin', 'super_admin')
)
AND permission_id IN (
    SELECT id FROM auth_permissions WHERE code IN ('qa:settings:read', 'qa:settings:write')
);

DELETE FROM auth_permissions
WHERE code IN ('qa:settings:read', 'qa:settings:write')
AND NOT EXISTS (
    SELECT 1
    FROM role_permissions
    WHERE role_permissions.permission_id = auth_permissions.id
);
