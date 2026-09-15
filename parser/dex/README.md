# Parser

## Descriptions
Parser extracts standardized transactions from raw transactions

## Terra chain packages

`parser.dex.chainId` picks the chain package through `dex.ChainNameOf` and
`parser/dex/dexwiring`, which is all conx and asi need. Terra takes a second step: the
factory address picks the contract generation, because terraswap migrated its factory
contract on terra classic. That second switch lives in
`parser/dex/terra/app_factory.go`, `parser/dex/srcstore/terra/factory.go` and
`cmd/parser/checkpoint/main.go`.

terra2 is phoenix-1, forked from terra classic. It inherited the same terraswap contracts,
so it shares classicv2's rules and its `classicV2ChainDataAdapter` from `pkg/dex/terra`.
`PiscoFactory` reaches the terra2 app but has no source store yet.

|                | classicv1                      | classicv2 / terra2                   |
| -------------- | ------------------------------ | ------------------------------------ |
| Factory        | `ClassicV1Factory`             | `ClassicV2Factory`, `MainnetFactory` |
| Address key    | `contract_address`             | `_contract_address`                  |
| Contract query | plain JSON, reads `res.Result` | base64, reads `res.Data`             |
| LCD package    | `pkg/terra/col4`               | `pkg/terra/cosmos45`                 |
| Extra rules    | none                           | classicv2 only: tax_payment, LP burn |

### col4 vs cosmos45

Both are response DTO sets plugged into the same generic LCD client in `pkg/terra/lcd`.
They differ only in what the node's REST API returns.

|                | `col4` (legacy amino)                                        | `cosmos45` (proto JSON)                                            |
| -------------- | ------------------------------------------------------------ | ------------------------------------------------------------------ |
| Contract state | `{"height", "result"}`, so the queried height can be verified | `{"data"}`, nothing to verify against                              |
| Query string   | plain JSON                                                   | base64                                                             |
| Tx sender      | `tx.value.msg[].value.sender`, `wasm/MsgExecuteContract`      | `tx.body.messages[].sender`, `/cosmwasm.wasm.v1.MsgExecuteContract` |

### Reading a block

Two switches, independent of each other:

| switch          | evaluated       | decides                                 |
| --------------- | --------------- | --------------------------------------- |
| factory address | once at startup | every row in the table above             |
| block height    | per block       | how a tx is read out of `block_results`  |

The LCD package never depends on height, so this is two states rather than a timeline:
`ClassicV2Factory` means `cosmos45` at every height that run parses. Nothing records the
migration height either; `ColumbusV1FactoryInstantiateHeight` (548,978) is unread and
`Columbus4EndHeight` (4,724,000) only bounds the legacy `cmd/collector/columbus4` run.

`rpc.RpcTxResultRes` carries both SDK shapes: `log`, a JSON string grouped per message,
before SDK 0.50, and `events`, one flat list per tx, from 0.50 on, where `msg_index`
arrives as an event attribute that `skipKeys` in `pkg/eventlog/finder.go` steps over.
Event types and attribute keys are unchanged across that boundary.

Each chain crosses it on its own terms:

- classic, `columbusCosmosSdk50StartHeight` (28,214,400) in `base_datastore.go`: below the
  boundary the grouped `log` is read as is, so `msg_index` survives.
- terra2, `cosmosSdk50StartHeight` (16,395,000) in `terra2_datastore.go`: both sides go
  through `convertEventsToRawTx`, and below the boundary the grouped `log` is flattened
  first, so `msg_index` is always dropped.

### Requires
- Filtered raw data store
- Data store to save parsed data


### Quick start

```zsh
git clone https://github.com/dezswap/cosmwasm-etl
cd cosmwasm-etl
vim config.yaml # to change environment variables.
export APP_TYPE=parser # one of (aggregator/collector/parser)
make up # starts App, with PostgresQL
make watch # server is automatically restarted when code is edited
# ...
make down # shut down all services
```


### Migrations

```zsh
make parser-migrate-up             # run migration
make parser-migrate-down           # down migration
make parser-migration-generate  # generate empty migration files
```


<!-- TODO: Add deployments, unit, e2e test  -->
