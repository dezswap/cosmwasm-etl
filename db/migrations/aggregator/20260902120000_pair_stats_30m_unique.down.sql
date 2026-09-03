-- Only the indexes are reversible. The duplicate rows the up migration removed are gone.
DROP INDEX CONCURRENTLY IF EXISTS pair_stats_30m_chain_id_timestamp_uidx;

CREATE INDEX CONCURRENTLY pair_stats_30m_chain_id_timestamp_uidx
    ON pair_stats_30m (chain_id, timestamp);

DROP INDEX CONCURRENTLY IF EXISTS pair_stats_30m_chain_id_timestamp_pair_id_uidx;
