# Contributing to CosmWasm ETL

## Branching

- Base all changes off `main`.
- Use descriptive branch names: `feat/short-description`, `fix/short-description`.

## Commit Messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/). Each commit message must have a type prefix:

| Prefix | When to use |
|---|---|
| `feat:` | New feature |
| `fix:` | Bug fix |
| `refactor:` | Code change that is not a feature or fix |
| `perf:` | Performance improvement |
| `docs:` | Documentation only |
| `test:` | Adding or updating tests |
| `chore:` | Maintenance (deps, tooling, config) |
| `ci:` | CI/CD changes |

`docs:`, `test:`, `chore:`, `ci:`, and `build:` commits are excluded from the changelog automatically.

## Pull Requests

1. Keep PRs focused — one concern per PR.
2. Fill in the PR template — background, summary, and checklist.
3. All CI checks (lint, test, build) must pass before merging.

## Development Setup

```bash
git clone https://github.com/dezswap/cosmwasm-etl.git
cd cosmwasm-etl

# Copy the example config and fill in your values
cp example.config.yaml config.yaml
vim config.yaml

# Start the app and local PostgreSQL via docker-compose
export APP_TYPE=collector  # one of: aggregator / collector / parser
make up

# Apply DB migrations
make parser-migrate-up
make aggregator-migrate-up
make collector-migrate-up
```

## Code Quality

```bash
make fmt        # format
make lint       # golangci-lint
make test       # unit tests (-short)
make cover      # tests with coverage
```

All CI checks must pass locally before opening a PR.

## Database Migrations

Migrations are timestamped SQL files under `db/migrations/<app>/`. Every schema change must include both an `.up.sql` and a `.down.sql`.

```bash
make parser-generate-migration   # create a new migration pair
make parser-migrate-up           # apply migrations
make parser-migrate-test         # run migration integration tests (-tags=mig,faker)
```

## Reporting Bugs & Feature Requests

Please use the [GitHub issue tracker](https://github.com/dezswap/cosmwasm-etl/issues).
