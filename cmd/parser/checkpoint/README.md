# Checkpoint

A CLI tool that creates snapshots of pool states at a specific block height.

## Overview

Checkpoint reads DEX pool states at a given block height and saves them to the database. It serves as a reference point when the parser needs to start synchronization from a specific block.

### Flow

1. Input target block height
2. Validate heights (DB checkpoint < target height ≤ node synced height)
3. Query pool states at the target height
4. Save checkpoint to database

## Usage

```bash
go run ./cmd/parser/checkpoint -height <block_height>
```

### Example

```bash
# Create checkpoint at block height 1000000
go run ./cmd/parser/checkpoint -height=1000000
```

## Configuration

Uses the parser section from `config.yaml`.
