-- +goose Up
-- Fix placeholder templates whose structure_json was seeded as a metadata object
-- instead of an outline array. The service requires structure_json to be a JSON array.
UPDATE report_templates
SET
    structure_json = CASE id
        WHEN '11111111-1111-4111-8111-111111111101'::uuid THEN '[
            {"title": "检查概况", "level": 1},
            {"title": "风险与问题", "level": 1},
            {"title": "整改建议", "level": 1}
        ]'::jsonb
        WHEN '11111111-1111-4111-8111-111111111102'::uuid THEN '[
            {"title": "审计概况", "level": 1},
            {"title": "库存核查", "level": 1},
            {"title": "审计结论", "level": 1}
        ]'::jsonb
        ELSE structure_json
    END,
    style_config_json = CASE id
        WHEN '11111111-1111-4111-8111-111111111101'::uuid THEN '{
            "styleProfileId": "first-slice-default-docx",
            "defaultFormat": "docx"
        }'::jsonb
        WHEN '11111111-1111-4111-8111-111111111102'::uuid THEN '{
            "styleProfileId": "first-slice-default-docx",
            "defaultFormat": "docx"
        }'::jsonb
        ELSE style_config_json
    END,
    updated_at = now()
WHERE id IN (
    '11111111-1111-4111-8111-111111111101'::uuid,
    '11111111-1111-4111-8111-111111111102'::uuid
)
  AND jsonb_typeof(structure_json) = 'object';

-- +goose Down
-- No-op: there is no safe way to restore the original metadata-as-structure format.
