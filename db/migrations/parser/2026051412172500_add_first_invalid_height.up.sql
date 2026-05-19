ALTER TABLE synced_height ADD COLUMN IF NOT EXISTS validation_height BIGINT NULL DEFAULT NULL;
COMMENT ON COLUMN synced_height.validation_height IS 'Next parser validation cursor. NULL means no validation is pending; a positive value means validate that height next and advance only after success.';
