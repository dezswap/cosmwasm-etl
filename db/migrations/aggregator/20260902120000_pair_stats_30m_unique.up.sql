-- pair_stats_30m had no unique key, so re-aggregating a window left a second row for
-- the same (chain_id, timestamp, pair_id) and consumers summing those rows counted the
-- window twice. Drop what reruns left behind, then make the key unique to upsert on.
--
-- golang-migrate splits this file on the semicolon with a plain byte scan, so no
-- comment here may contain one and no statement may be a DO block.
--
-- Run with the aggregator stopped. An old-code writer inserts duplicates that fail the
-- index build, and new-code writes fail with 42P10 until the index exists.

DELETE FROM pair_stats_30m ps
WHERE EXISTS (
    SELECT 1
    FROM pair_stats_30m newer
    WHERE newer.chain_id = ps.chain_id
      AND newer.timestamp = ps.timestamp
      AND newer.pair_id = ps.pair_id
      AND newer.id > ps.id
);

-- A failed CONCURRENTLY build leaves an invalid index behind, which IF NOT EXISTS would
-- skip past on a retry.
DROP INDEX CONCURRENTLY IF EXISTS pair_stats_30m_chain_id_timestamp_pair_id_uidx;

CREATE UNIQUE INDEX CONCURRENTLY pair_stats_30m_chain_id_timestamp_pair_id_uidx
    ON pair_stats_30m (chain_id, timestamp, pair_id);

-- A left prefix of the index above, and so redundant now.
DROP INDEX CONCURRENTLY IF EXISTS pair_stats_30m_chain_id_timestamp_uidx;
