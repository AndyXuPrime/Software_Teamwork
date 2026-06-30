\connect auth_system

INSERT INTO auth_users (
    id,
    username,
    display_name,
    email,
    status,
    created_at,
    updated_at
)
VALUES (
    'usr_local_admin',
    'admin',
    'Local Demo Administrator',
    'admin@example.invalid',
    'active',
    now(),
    now()
)
ON CONFLICT (username) WHERE deleted_at IS NULL DO UPDATE
SET display_name = EXCLUDED.display_name,
    email = EXCLUDED.email,
    status = EXCLUDED.status,
    updated_at = now();

INSERT INTO auth_credentials (
    id,
    user_id,
    credential_type,
    password_hash,
    password_hash_alg,
    password_hash_params_version,
    password_hash_params_json,
    password_changed_at,
    created_at,
    updated_at
)
VALUES (
    'cred_local_admin_password',
    'usr_local_admin',
    'password',
    '$argon2id$v=19$m=65536,t=3,p=2$bG9jYWwtZGVtby1zYWx0IQ$tESTl/LqUlaDlE8hP4+CNLG5go/+X2xvYXBdqk+4eOI',
    'argon2id',
    'argon2id-v1',
    '{"memoryKiB":65536,"iterations":3,"parallelism":2,"saltBytes":16,"keyBytes":32}'::jsonb,
    now(),
    now(),
    now()
)
ON CONFLICT (user_id, credential_type) DO UPDATE
SET password_hash = EXCLUDED.password_hash,
    password_hash_alg = EXCLUDED.password_hash_alg,
    password_hash_params_version = EXCLUDED.password_hash_params_version,
    password_hash_params_json = EXCLUDED.password_hash_params_json,
    password_changed_at = now(),
    updated_at = now();

INSERT INTO user_roles (
    id,
    user_id,
    role_id,
    assigned_by,
    assigned_at,
    created_at
)
SELECT
    'urole_local_admin_admin',
    'usr_local_admin',
    r.id,
    'local-seed',
    now(),
    now()
FROM auth_roles r
WHERE r.code = 'admin'
ON CONFLICT (user_id, role_id) DO NOTHING;

\connect knowledge_system

INSERT INTO knowledge_bases (
    id,
    name,
    description,
    doc_type,
    chunk_strategy,
    retrieval_strategy,
    created_by,
    created_at,
    updated_at
)
VALUES (
    'kb_local_demo',
    'Local Demo Knowledge Base',
    'Seed knowledge base for local integration smoke tests.',
    'GENERAL',
    '{"chunkSize":800,"overlap":120}'::jsonb,
    '{"topK":5,"scoreThreshold":0.2}'::jsonb,
    'usr_local_admin',
    now(),
    now()
)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    doc_type = EXCLUDED.doc_type,
    chunk_strategy = EXCLUDED.chunk_strategy,
    retrieval_strategy = EXCLUDED.retrieval_strategy,
    updated_at = now();

\connect document_system

INSERT INTO report_types (code, name, description, enabled, updated_at)
VALUES
    ('summer_peak_inspection', 'Summer Peak Inspection Report', 'Local demo report type for peak-season inspection workflows.', true, now()),
    ('coal_inventory_audit', 'Coal Inventory Audit Report', 'Local demo report type for coal inventory audit workflows.', true, now())
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    enabled = EXCLUDED.enabled,
    updated_at = now();
