BEGIN;

DROP TABLE IF EXISTS "fcd_tx_log";

DROP INDEX IF EXISTS fcd_tx_log_timestamp_idx;
DROP INDEX IF EXISTS fcd_tx_log_height_idx;
DROP INDEX IF EXISTS fcd_tx_log_height_hash_key;
DROP INDEX IF EXISTS fcd_tx_log_hash_idx;
DROP INDEX IF EXISTS fcd_tx_log_fcd_offset_idx;
DROP INDEX IF EXISTS fcd_tx_log_address_idx;

COMMIT;
