-- +goose Up
INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT 'rperm_standard_document_upload_v2', r.id, p.id, now()
FROM auth_roles r, auth_permissions p
WHERE r.code = 'standard' AND p.code = 'document:upload'
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp2
    WHERE rp2.role_id = r.id AND rp2.permission_id = p.id
  );

INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT 'rperm_standard_report_write_v2', r.id, p.id, now()
FROM auth_roles r, auth_permissions p
WHERE r.code = 'standard' AND p.code = 'report:write'
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp2
    WHERE rp2.role_id = r.id AND rp2.permission_id = p.id
  );

INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT 'rperm_admin_system_admin_v2', r.id, p.id, now()
FROM auth_roles r, auth_permissions p
WHERE r.code = 'admin' AND p.code = 'system:admin'
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp2
    WHERE rp2.role_id = r.id AND rp2.permission_id = p.id
  );

-- +goose Down
DELETE FROM role_permissions
WHERE id IN (
  'rperm_standard_document_upload_v2',
  'rperm_standard_report_write_v2',
  'rperm_admin_system_admin_v2'
);
