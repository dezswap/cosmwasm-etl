-- Roll back this index before reverting the token_exception schema migration.
DROP INDEX CONCURRENTLY IF EXISTS token_exception_chain_id_hidden_idx;
