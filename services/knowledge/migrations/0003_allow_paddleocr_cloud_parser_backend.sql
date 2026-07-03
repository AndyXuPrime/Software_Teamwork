-- +goose Up
ALTER TABLE parser_configs DROP CONSTRAINT IF EXISTS parser_configs_backend_check;
ALTER TABLE parser_configs ADD CONSTRAINT parser_configs_backend_check CHECK (backend IN ('builtin', 'tika', 'unstructured', 'local_ocr', 'remote_compatible', 'paddleocr_cloud'));

-- +goose Down
UPDATE parser_configs
SET
  backend = 'local_ocr',
  endpoint_url = NULL,
  default_parameters = default_parameters - 'paddleocr_base_url' - 'paddleocr_access_token' - 'paddleocr_algorithm'
WHERE backend = 'paddleocr_cloud';

ALTER TABLE parser_configs DROP CONSTRAINT IF EXISTS parser_configs_backend_check;
ALTER TABLE parser_configs ADD CONSTRAINT parser_configs_backend_check CHECK (backend IN ('builtin', 'tika', 'unstructured', 'local_ocr', 'remote_compatible'));
