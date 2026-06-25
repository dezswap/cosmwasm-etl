CREATE INDEX CONCURRENTLY IF NOT EXISTS price_chain_id_token_id_price_token_id_height_idx
    ON price (chain_id, token_id, price_token_id, height DESC);
