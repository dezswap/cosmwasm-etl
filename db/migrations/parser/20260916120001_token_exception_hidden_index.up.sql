-- Keep concurrent index DDL in its own migration. The parser runner executes each
-- file as one query to preserve historical DO blocks containing semicolons.
CREATE INDEX CONCURRENTLY IF NOT EXISTS token_exception_chain_id_hidden_idx
    ON "token_exception" ("chain_id", "hidden");
