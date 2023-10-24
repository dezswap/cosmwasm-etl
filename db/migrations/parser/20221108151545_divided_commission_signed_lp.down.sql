BEGIN;

ALTER TABLE parsed_tx DROP COLUMN commission0_amount;
ALTER TABLE parsed_tx DROP COLUMN commission1_amount;

-- rollback negative lp amount for withdraw
UPDATE
  parsed_tx
SET
  lp_amount = -1 * lp_amount,
  asset0_amount = -1 * asset0_amount,
  asset1_amount = -1 * asset1_amount
WHERE
  parsed_tx."type" = 'withdraw'
  AND parsed_tx.lp_amount < 0
  AND parsed_tx.asset0_amount < 0
  AND parsed_tx.asset1_amount < 0;

COMMIT;
