DROP INDEX CONCURRENTLY IF EXISTS account_stats_30m_chain_id_account_id_pair_id_idx;
DROP INDEX CONCURRENTLY IF EXISTS account_stats_30m_chain_id_account_id_timestamp_idx;

BEGIN;

ALTER TABLE account_stats_30m
    DROP COLUMN IF EXISTS net_lp_amount,
    DROP COLUMN IF EXISTS net_asset1_amount,
    DROP COLUMN IF EXISTS net_asset0_amount,
    DROP COLUMN IF EXISTS price_token,
    DROP COLUMN IF EXISTS net_flow_in_price,
    DROP COLUMN IF EXISTS withdraw_value_in_price,
    DROP COLUMN IF EXISTS provide_value_in_price,
    DROP COLUMN IF EXISTS swap_volume_in_price,
    DROP COLUMN IF EXISTS withdraw_tx_cnt,
    DROP COLUMN IF EXISTS provide_tx_cnt,
    DROP COLUMN IF EXISTS swap_tx_cnt;

COMMIT;
