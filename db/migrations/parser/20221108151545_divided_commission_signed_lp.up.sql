BEGIN;
-- negative lp amount for withdraw
UPDATE
  parsed_tx
SET
  lp_amount = -1 * lp_amount,
  asset0_amount = -1 * asset0_amount,
  asset1_amount = -1 * asset1_amount
WHERE
  parsed_tx."type" = 'withdraw'
  AND parsed_tx.lp_amount > 0
  AND parsed_tx.asset0_amount > 0
  AND parsed_tx.asset1_amount > 0;

-- divide commission
ALTER TABLE parsed_tx ADD COLUMN commission0_amount DECIMAL(40);
ALTER TABLE parsed_tx ADD COLUMN commission1_amount DECIMAL(40);

-- commission 0
UPDATE
  parsed_tx
SET
  commission0_amount = parsed_tx.commission_amount
WHERE
  parsed_tx."type" = 'swap'
  AND parsed_tx.asset0_amount < 0;

-- commission 1
UPDATE
  parsed_tx
SET
  commission1_amount = parsed_tx.commission_amount
WHERE
  parsed_tx."type" = 'swap'
  AND parsed_tx.asset1_amount < 0;

COMMIT;
