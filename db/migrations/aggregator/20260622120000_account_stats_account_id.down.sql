DROP INDEX CONCURRENTLY IF EXISTS account_stats_30m_chain_id_account_id_idx;
DROP INDEX CONCURRENTLY IF EXISTS account_stats_30m_chain_id_timestamp_account_id_pair_id_uidx;

BEGIN;

ALTER TABLE account_stats_30m
    DROP CONSTRAINT IF EXISTS account_stats_30m_account_id_not_null;

ALTER TABLE account_stats_30m
    DROP COLUMN IF EXISTS account_id;

COMMIT;
