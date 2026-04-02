CREATE INDEX parsed_tx_type_timestamp_idx ON parsed_tx (type, timestamp DESC);

CREATE INDEX parsed_tx_contract_type_timestamp_idx ON parsed_tx (contract, type, timestamp DESC);

CREATE INDEX parsed_tx_asset0_type_timestamp_idx ON parsed_tx (asset0, type, timestamp DESC);
CREATE INDEX parsed_tx_asset1_type_timestamp_idx ON parsed_tx (asset1, type, timestamp DESC);