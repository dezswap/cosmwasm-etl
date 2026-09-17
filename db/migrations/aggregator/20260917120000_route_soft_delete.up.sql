BEGIN;

-- Routes reaching a hidden token are retired rather than removed: price rows carry a
-- route_id and are never rewritten. Readers must filter on deleted_at is null.
ALTER TABLE route ADD COLUMN IF NOT EXISTS deleted_at DOUBLE PRECISION;

COMMIT;

CREATE INDEX CONCURRENTLY IF NOT EXISTS route_chain_id_deleted_at_idx
    ON route (chain_id) WHERE deleted_at IS NOT NULL;
