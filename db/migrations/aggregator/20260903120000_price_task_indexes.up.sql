-- LatestRouteUpdateTimestamp runs max(created_at) once per height, and created_at is
-- in no index, so every height reads the whole chain partition of route.
CREATE INDEX CONCURRENTLY IF NOT EXISTS route_chain_id_created_at_idx
    ON route (chain_id, created_at DESC);

-- Liquidity reads one pair's latest lp_history row at or below a height, once per hop
-- of every route priced. A btree stops at the first range, so pair_id must lead height.
CREATE INDEX CONCURRENTLY IF NOT EXISTS lp_history_chain_id_pair_id_height_idx
    ON lp_history (chain_id, pair_id, height DESC);
