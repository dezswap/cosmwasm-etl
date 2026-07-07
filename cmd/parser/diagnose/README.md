# Parser Diagnose

`parser-diagnose` replays parser logic for a height range and pair contract without advancing parser synced height or writing parsed rows.

```bash
make parser-diagnose
./build/parser-diagnose --from 26407001 --to 26408000 --contract terra1...
```

Use this command after a `parser.pool_validation_failed` log. The log includes:

- `chain_id`
- `contract`
- `validation_height`
- `investigation_from_height`
- `investigation_to_height`
- `validation_interval`
- `mismatch_type`
- `actual`
- `expected`
- `lookup_tables`

## Investigation Query Templates

Check quarantined transactions in the investigation window. `contract` can be a token contract, so scan `raw_tx` for the pair contract or expected action.

```sql
SELECT *
FROM parse_quarantine
WHERE chain_id = $1
  AND height BETWEEN $2 AND $3
ORDER BY height ASC, id ASC;
```

Check parsed transactions for the pair in the same window.

```sql
SELECT height, hash, type, sender, asset0_amount, asset1_amount,
       lp_amount, commission0_amount, commission1_amount
FROM parsed_tx
WHERE chain_id = $1
  AND contract = $2
  AND height BETWEEN $3 AND $4
ORDER BY height DESC;
```

Find raw collector transactions when collector source tables are available.

```sql
SELECT *
FROM collector_blocks
WHERE chain_id = $1
  AND height BETWEEN $2 AND $3
  AND txs::text ILIKE '%' || $4 || '%'
ORDER BY height ASC;
```

Find raw FCD logs when `fcd_tx_log` is available.

```sql
SELECT *
FROM fcd_tx_log
WHERE address = $1
  AND height BETWEEN $2 AND $3
ORDER BY height DESC, fcd_offset DESC;
```
