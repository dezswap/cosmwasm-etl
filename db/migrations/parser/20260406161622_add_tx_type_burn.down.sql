UPDATE parsed_tx SET type = 'transfer' WHERE type = 'lp_burn';

CREATE TYPE tx_type_old AS ENUM ('create_pair', 'swap', 'provide', 'withdraw', 'initial_provide', 'transfer');

ALTER TABLE parsed_tx ALTER COLUMN type TYPE tx_type_old USING type::text::tx_type_old;
DROP TYPE tx_type;
ALTER TYPE tx_type_old RENAME TO tx_type;