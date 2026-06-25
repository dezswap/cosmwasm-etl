BEGIN;

ALTER TABLE account_stats_30m
    ADD COLUMN IF NOT EXISTS swap_tx_cnt bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS provide_tx_cnt bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS withdraw_tx_cnt bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS swap_volume_in_price numeric NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS provide_value_in_price numeric NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS withdraw_value_in_price numeric NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS net_flow_in_price numeric NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS price_token varchar NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS net_asset0_amount numeric NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS net_asset1_amount numeric NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS net_lp_amount numeric NOT NULL DEFAULT 0;

COMMIT;

CREATE INDEX CONCURRENTLY IF NOT EXISTS account_stats_30m_chain_id_account_id_timestamp_idx
    ON account_stats_30m (chain_id, account_id, timestamp);

CREATE INDEX CONCURRENTLY IF NOT EXISTS account_stats_30m_chain_id_account_id_pair_id_idx
    ON account_stats_30m (chain_id, account_id, pair_id);
