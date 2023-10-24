# Parser

## Descriptions
Parser extracts standardized transactions from raw transactions

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
