-- +goose Up
-- Upgrade already-applied placeholder report templates from the old first-slice
-- outline arrays to the richer Chinese business outlines introduced later.

UPDATE report_types
SET
    name = CASE code
        WHEN 'summer_peak_inspection' THEN '迎峰度夏检查报告'
        WHEN 'coal_inventory_audit' THEN '煤库存审计报告'
        ELSE name
    END,
    description = CASE code
        WHEN 'summer_peak_inspection' THEN '迎峰度夏检查报告'
        WHEN 'coal_inventory_audit' THEN '煤库存审计报告'
        ELSE description
    END,
    updated_at = now()
WHERE (code = 'summer_peak_inspection' AND name = '迎峰度夏检查报告' AND description = '迎峰度夏检查报告')
   OR (code = 'coal_inventory_audit' AND name = '煤库存审计报告' AND description = '煤库存审计报告');

UPDATE report_templates
SET
    template_name = '迎峰度夏检查报告占位模板',
    description = 'Needs Decision: formal DOCX template file is pending. Placeholder seeded from services/document/migrations/0003_seed_initial_report_defaults.sql.',
    structure_json = '[
        {"title": "报告摘要与检查结论", "level": 1},
        {"title": "检查范围与依据", "level": 1},
        {"title": "供电负荷与设备运行核查", "level": 1},
        {"title": "防汛防暑与应急保障", "level": 1},
        {"title": "隐患问题与整改闭环", "level": 1},
        {"title": "后续保障建议", "level": 1}
    ]'::jsonb,
    style_config_json = '{
        "styleProfileId": "first-slice-default-docx",
        "defaultFormat": "docx"
    }'::jsonb,
    updated_at = now()
WHERE id = '11111111-1111-4111-8111-111111111101'::uuid
  AND report_type = 'summer_peak_inspection'
  AND filename = 'placeholder-summer-peak-inspection.docx'
  AND created_by = 'system'
  AND (
      jsonb_typeof(structure_json) = 'object'
      OR structure_json = '[
          {"title": "检查概况", "level": 1},
          {"title": "风险与问题", "level": 1},
          {"title": "整改建议", "level": 1}
      ]'::jsonb
  );

UPDATE report_templates
SET
    template_name = '煤库存审计报告占位模板',
    description = 'Needs Decision: formal DOCX template file is pending. Placeholder seeded from services/document/migrations/0003_seed_initial_report_defaults.sql.',
    structure_json = '[
        {"title": "审计范围与依据", "level": 1},
        {"title": "库存账实核查", "level": 1},
        {"title": "煤质与计量抽查", "level": 1},
        {"title": "库存周转与保供风险", "level": 1},
        {"title": "问题清单与整改建议", "level": 1},
        {"title": "审计结论", "level": 1}
    ]'::jsonb,
    style_config_json = '{
        "styleProfileId": "first-slice-default-docx",
        "defaultFormat": "docx"
    }'::jsonb,
    updated_at = now()
WHERE id = '11111111-1111-4111-8111-111111111102'::uuid
  AND report_type = 'coal_inventory_audit'
  AND filename = 'placeholder-coal-inventory-audit.docx'
  AND created_by = 'system'
  AND (
      jsonb_typeof(structure_json) = 'object'
      OR structure_json = '[
          {"title": "审计概况", "level": 1},
          {"title": "库存核查", "level": 1},
          {"title": "审计结论", "level": 1}
      ]'::jsonb
  );

-- +goose Down
-- No-op: the forward migration only upgrades known default placeholder rows.
