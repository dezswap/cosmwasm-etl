BEGIN;

CREATE TABLE "synced_height" (
  "id" BIGSERIAL NOT NULL PRIMARY KEY,
  "chain_id" VARCHAR NOT NULL, CHECK("chain_id" <> ''),
  "height" BIGINT NOT NULL, CHECK("height" >= 0),
  "created_at" DOUBLE PRECISION NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE TYPE tx_type as ENUM('create_pair', 'swap','provide','withdraw','mint_token', 'transfer');

CREATE TABLE "pair"(
  "id" BIGSERIAL NOT NULL PRIMARY KEY,
  "chain_id" VARCHAR NOT NULL, CHECK("chain_id" <> ''),
  "contract" VARCHAR NOT NULL, CHECK("contract" <> ''),
  "asset0" VARCHAR NOT NULL, CHECK("asset0" <> ''),
  "asset1" VARCHAR NOT NULL, CHECK("asset1" <> ''),
  "lp" VARCHAR NOT NULL, CHECK("lp" <> ''),
  "created_at" DOUBLE PRECISION NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  "meta" JSON
);

CREATE UNIQUE INDEX pair_chain_id_contract_key ON pair ("chain_id", "contract");
CREATE UNIQUE INDEX pair_chain_id_asset0_asset1_key ON pair ("chain_id", "asset0", "asset1");
CREATE UNIQUE INDEX pair_chain_id_lp_key ON pair ("chain_id","lp");

CREATE TABLE "parsed_tx"(
  "id" BIGSERIAL NOT NULL PRIMARY KEY,
  "chain_id" VARCHAR NOT NULL, CHECK("chain_id" <> ''),
  "height" BIGINT NOT NULL,
  "timestamp" DOUBLE PRECISION NOT NULL,
  "hash" VARCHAR NOT NULL, CHECK("hash" <> ''),
  "type" tx_type NOT NULL,
  "sender" VARCHAR NOT NULL, CHECK("sender" <> ''),
  "contract" VARCHAR NOT NULL, CHECK("contract" <> ''),
  "asset0" VARCHAR,
  "asset0_amount" DECIMAL(40) NOT NULL,
  "asset1" VARCHAR,
  "asset1_amount" DECIMAL(40) NOT NULL,
  "lp" VARCHAR,
  "lp_amount" DECIMAL(40) NOT NULL,
  "commission_amount" DECIMAL(40),
  "created_at" DOUBLE PRECISION NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  "meta" JSON
);

CREATE INDEX parsed_tx_chain_id_height_idx ON parsed_tx ("chain_id", "height");
CREATE INDEX parsed_tx_contract_idx ON parsed_tx ("contract");
CREATE INDEX parsed_tx_sender_idx ON parsed_tx ("sender");
CREATE INDEX parsed_tx_timestamp_idx ON parsed_tx ("timestamp");
ALTER TABLE parsed_tx ADD CONSTRAINT parsed_tx_contract_pair_fkey FOREIGN KEY ("chain_id","contract") REFERENCES pair ("chain_id", "contract");

CREATE TABLE "pool_info"(
  "id" BIGSERIAL NOT NULL PRIMARY KEY,
  "chain_id" VARCHAR NOT NULL, CHECK("chain_id" <> ''),
  "height" BIGINT NOT NULL,
  "contract" VARCHAR NOT NULL, CHECK("contract" <> ''),
  "asset0_amount" DECIMAL(40) NOT NULL,
  "asset1_amount" DECIMAL(40) NOT NULL,
  "lp_amount" DECIMAL(40) NOT NULL,
  "created_at" DOUBLE PRECISION NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  "meta" JSON
);

CREATE INDEX pool_info_chain_id_height_idx ON pool_info ("chain_id", "height");
CREATE INDEX pool_info_contract_idx ON pool_info ("contract");
CREATE UNIQUE INDEX pool_info_chain_id_height_contract_key ON pool_info ("chain_id", "height", "contract");
ALTER TABLE pool_info ADD CONSTRAINT pool_info_contract_pair_fkey FOREIGN KEY ("chain_id","contract") REFERENCES pair ("chain_id", "contract");

CREATE TABLE IF NOT EXISTS "tokens" (
  "id"         BIGSERIAL PRIMARY KEY,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE,
  "chain_id"   TEXT NOT NULL,
  "address"    TEXT NOT NULL,
  "protocol"   TEXT,
  "symbol"     TEXT,
  "name"       TEXT,
  "decimals"   SMALLINT,
  "icon"       TEXT,
  "verified"   BOOLEAN DEFAULT FALSE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tokens_deleted_at ON tokens ("deleted_at");
CREATE INDEX IF NOT EXISTS idx_tokens_address ON tokens ("address");
CREATE UNIQUE INDEX IF NOT EXISTS idx_tokens_chain_id_address_key ON tokens ("chain_id", "address");
CREATE INDEX IF NOT EXISTS idx_tokens_chain_id ON tokens ("chain_id");


CREATE TABLE IF NOT EXISTS "latest_pools" (
  "id"            BIGSERIAL PRIMARY KEY,
  "created_at"    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at"    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at"    TIMESTAMP WITH TIME ZONE,
  "chain_id"      TEXT NOT NULL,
  "address"       TEXT NOT NULL,
  "height"        BIGINT,
  "asset0"        TEXT,
  "asset0_amount" TEXT,
  "asset1"        TEXT,
  "asset1_amount" TEXT,
  "lp"            TEXT,
  "lp_amount"     TEXT
);

CREATE INDEX IF NOT EXISTS idx_latest_pools_address ON latest_pools ("address");
CREATE UNIQUE INDEX IF NOT EXISTS idx_latest_pools_chain_id_address_key ON latest_pools ("chain_id", "address");
CREATE INDEX IF NOT EXISTS idx_latest_pools_chain_id ON latest_pools ("chain_id");
CREATE INDEX IF NOT EXISTS idx_latest_pools_deleted_at ON latest_pools ("deleted_at");
CREATE INDEX IF NOT EXISTS idx_latest_pools_lp ON latest_pools ("lp");
CREATE INDEX IF NOT EXISTS idx_latest_pools_asset1 ON latest_pools ("asset1");
CREATE INDEX IF NOT EXISTS idx_latest_pools_asset0 ON latest_pools ("asset0");

COMMIT;
