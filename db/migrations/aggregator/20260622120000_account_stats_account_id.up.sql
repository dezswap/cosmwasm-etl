BEGIN;

ALTER TABLE account_stats_30m
    ADD COLUMN IF NOT EXISTS account_id bigint;

INSERT INTO account(address)
SELECT DISTINCT address
FROM account_stats_30m
WHERE address IS NOT NULL
  AND address <> ''
ON CONFLICT DO NOTHING;

UPDATE account_stats_30m ast
SET account_id = a.id
FROM account a
WHERE ast.account_id IS NULL
  AND ast.address = a.address;

DELETE FROM account_stats_30m ast
USING (
    SELECT id
    FROM (
        SELECT id,
               row_number() OVER (
                   PARTITION BY chain_id, timestamp, account_id, pair_id
                   ORDER BY modified_at DESC, id DESC
               ) AS rn
        FROM account_stats_30m
        WHERE account_id IS NOT NULL
    ) ranked
    WHERE rn > 1
) dups
WHERE ast.id = dups.id;

ALTER TABLE account_stats_30m
    DROP CONSTRAINT IF EXISTS account_stats_30m_account_id_not_null;

-- Validate with a temporary CHECK first so SET NOT NULL can avoid a long table scan/lock.
ALTER TABLE account_stats_30m
    ADD CONSTRAINT account_stats_30m_account_id_not_null
        CHECK (account_id IS NOT NULL) NOT VALID;

COMMIT;

ALTER TABLE account_stats_30m
    VALIDATE CONSTRAINT account_stats_30m_account_id_not_null;

ALTER TABLE account_stats_30m
    ALTER COLUMN account_id SET NOT NULL;

ALTER TABLE account_stats_30m
    DROP CONSTRAINT IF EXISTS account_stats_30m_account_id_not_null;

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS account_stats_30m_chain_id_timestamp_account_id_pair_id_uidx
    ON account_stats_30m (chain_id, timestamp, account_id, pair_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS account_stats_30m_chain_id_account_id_idx
    ON account_stats_30m (chain_id, account_id);
