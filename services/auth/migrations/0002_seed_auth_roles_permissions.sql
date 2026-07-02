-- +goose Up
INSERT INTO auth_roles (id, code, name, description, enabled, system_role, created_at, updated_at)
VALUES
    ('role_standard', 'standard', 'Standard User', 'Default role for newly created users.', TRUE, TRUE, now(), now()),
    ('role_admin', 'admin', 'Administrator', 'Administrator role for operational management.', TRUE, TRUE, now(), now()),
    ('role_super_admin', 'super_admin', 'Super Administrator', 'Full platform administration role.', TRUE, TRUE, now(), now())
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    enabled = EXCLUDED.enabled,
    system_role = EXCLUDED.system_role,
    updated_at = now();

INSERT INTO auth_permissions (id, code, domain, action, description, enabled, created_at, updated_at)
VALUES
    ('perm_knowledge_read', 'knowledge:read', 'knowledge', 'read', 'Read knowledge resources.', TRUE, now(), now()),
    ('perm_knowledge_write', 'knowledge:write', 'knowledge', 'write', 'Create and modify knowledge resources.', TRUE, now(), now()),
    ('perm_document_read', 'document:read', 'document', 'read', 'Read document resources.', TRUE, now(), now()),
    ('perm_document_upload', 'document:upload', 'document', 'upload', 'Upload document resources.', TRUE, now(), now()),
    ('perm_document_update', 'document:update', 'document', 'update', 'Update document resources.', TRUE, now(), now()),
    ('perm_document_delete', 'document:delete', 'document', 'delete', 'Delete document resources.', TRUE, now(), now()),
    ('perm_report_read', 'report:read', 'report', 'read', 'Read report resources.', TRUE, now(), now()),
    ('perm_report_write', 'report:write', 'report', 'write', 'Create and modify report resources.', TRUE, now(), now()),
    ('perm_qa_use', 'qa:use', 'qa', 'use', 'Use QA capabilities.', TRUE, now(), now()),
    ('perm_admin_model_profile_write', 'admin:model-profile:write', 'admin', 'model-profile:write', 'Manage runtime model profiles.', TRUE, now(), now()),
    ('perm_admin_parser_config_write', 'admin:parser-config:write', 'admin', 'parser-config:write', 'Manage parser configurations.', TRUE, now(), now()),
    ('perm_system_admin', 'system:admin', 'system', 'admin', 'Administer system-level settings.', TRUE, now(), now())
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
        ('rperm_standard_knowledge_read', 'standard', 'knowledge:read'),
        ('rperm_standard_document_read', 'standard', 'document:read'),
        ('rperm_standard_report_read', 'standard', 'report:read'),
        ('rperm_standard_qa_use', 'standard', 'qa:use'),
        ('rperm_admin_knowledge_read', 'admin', 'knowledge:read'),
        ('rperm_admin_knowledge_write', 'admin', 'knowledge:write'),
        ('rperm_admin_document_read', 'admin', 'document:read'),
        ('rperm_admin_document_upload', 'admin', 'document:upload'),
        ('rperm_admin_document_update', 'admin', 'document:update'),
        ('rperm_admin_document_delete', 'admin', 'document:delete'),
        ('rperm_admin_report_read', 'admin', 'report:read'),
        ('rperm_admin_report_write', 'admin', 'report:write'),
        ('rperm_admin_qa_use', 'admin', 'qa:use'),
        ('rperm_admin_model_profile_write', 'admin', 'admin:model-profile:write'),
        ('rperm_admin_parser_config_write', 'admin', 'admin:parser-config:write'),
        ('rperm_super_knowledge_read', 'super_admin', 'knowledge:read'),
        ('rperm_super_knowledge_write', 'super_admin', 'knowledge:write'),
        ('rperm_super_document_read', 'super_admin', 'document:read'),
        ('rperm_super_document_upload', 'super_admin', 'document:upload'),
        ('rperm_super_document_update', 'super_admin', 'document:update'),
        ('rperm_super_document_delete', 'super_admin', 'document:delete'),
        ('rperm_super_report_read', 'super_admin', 'report:read'),
        ('rperm_super_report_write', 'super_admin', 'report:write'),
        ('rperm_super_qa_use', 'super_admin', 'qa:use'),
        ('rperm_super_model_profile_write', 'super_admin', 'admin:model-profile:write'),
        ('rperm_super_parser_config_write', 'super_admin', 'admin:parser-config:write'),
        ('rperm_super_system_admin', 'super_admin', 'system:admin')
) AS seed(id, role_code, permission_code)
INNER JOIN auth_roles r ON r.code = seed.role_code
INNER JOIN auth_permissions p ON p.code = seed.permission_code
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- +goose Down
DELETE FROM user_roles
WHERE role_id IN (
    SELECT id FROM auth_roles WHERE code IN ('standard', 'admin', 'super_admin')
);

DELETE FROM role_permissions
WHERE role_id IN (
    SELECT id FROM auth_roles WHERE code IN ('standard', 'admin', 'super_admin')
);

DELETE FROM auth_roles
WHERE code IN ('standard', 'admin', 'super_admin');

DELETE FROM auth_permissions
WHERE code IN (
    'knowledge:read',
    'knowledge:write',
    'document:read',
    'document:upload',
    'document:update',
    'document:delete',
    'report:read',
    'report:write',
    'qa:use',
    'admin:model-profile:write',
    'admin:parser-config:write',
    'system:admin'
);
