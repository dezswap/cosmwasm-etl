# CosmWasm ETL

## Getting Started

### Requirements

- [Go installation](https://golang.org/dl/) (preferably v1.18) and a [correctly configured path](https://golang.org/doc/install#install).
- A local [docker, docker-compose](https://docs.docker.com)
- [Golangci-lint](https://github.com/golangci/golangci-lint) to improve code quality


### Quick start

```zsh
git clone https://github.com/dezswap/cosmwasm-etl
cd cosmwasm-etl
vim config.yaml # to change environment variables.
export APP_TYPE=collector # one of (aggregator/collector/parser)
make up # starts App, with PostgresQL
make watch # server is automatically restarted when code is edited
# ...
make down # shut down all services
```

## Commands

### Run

```zsh
make all          # Test and build all apps
make deps         # Download dependencies
make watch        # Run development server, recompile and run tests on file change
make clean        # Remove compiled binaries from local disk
make fmt          # Format code
make lint         # Run golangci-lint on code changed since forked from master branch
make prune-deps   # Remove unused dependencies from go.mod & go.sum
make image        # Create docker image with minimal binary
make build-docker # Build with special params to create a complete binary,
                  # see Dockerfile
make build-all    # Build all apps
```

### Test

```zsh
make test       # Run tests for all packages
make cover      # Check coverage for all packages
```

## Packages

## Contributing

### Bug Reports & Feature Requests

Please use the [issue tracker](https://github.com/dezswap/cosmwasm-etl/issues) to report any bugs or ask feature requests.
