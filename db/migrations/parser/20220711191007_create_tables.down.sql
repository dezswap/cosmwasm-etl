BEGIN;

DROP TABLE IF EXISTS "synced_height";

DROP INDEX IF EXISTS parsed_tx_chain_id_height_idx;
DROP INDEX IF EXISTS parsed_tx_contract_idx;
DROP INDEX IF EXISTS parsed_tx_sender_idx;
DROP TABLE IF EXISTS "parsed_tx";
DROP TYPE IF EXISTS tx_type;

DROP INDEX IF EXISTS pool_info_chain_id_height_idx;
DROP INDEX IF EXISTS pool_info_contract_idx;
DROP INDEX IF EXISTS pool_info_chain_id_height_contract_key;
DROP TABLE IF EXISTS "pool_info";

DROP INDEX IF EXISTS pair_chain_id_contract_key;
DROP INDEX IF EXISTS pair_chain_id_asset_0_asset_1_key;
DROP INDEX IF EXISTS pair_chain_id_lp_key;
DROP TABLE IF EXISTS "pair";

COMMIT;
