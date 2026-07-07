package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/parser/dex/dexwiring"
	"github.com/dezswap/cosmwasm-etl/parser/dex/repo"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

type diagnosisReport struct {
	ChainID  string              `json:"chain_id"`
	From     uint64              `json:"from_height"`
	To       uint64              `json:"to_height"`
	Contract string              `json:"contract"`
	Results  []diagnosisTxResult `json:"results"`
}

type diagnosisTxResult struct {
	Height        uint64           `json:"height"`
	Hash          string           `json:"hash"`
	Sender        string           `json:"sender"`
	ParsedTxCount int              `json:"parsed_tx_count"`
	ParsedTxs     []p_dex.ParsedTx `json:"parsed_txs,omitempty"`
	Error         string           `json:"error,omitempty"`
}

func main() {
	from := flag.Uint64("from", 0, "first height to diagnose")
	to := flag.Uint64("to", 0, "last height to diagnose")
	contract := flag.String("contract", "", "pair contract address to diagnose")
	flag.Parse()

	if *from == 0 || *to == 0 || *contract == "" {
		fail("required flags: --from, --to, --contract")
	}
	if *from > *to {
		fail("--from must be less than or equal to --to")
	}

	c := configs.New()
	dc := c.Parser.DexConfig
	if err := dc.Validate(); err != nil {
		fail(fmt.Sprintf("invalid parser dex config: %s", err))
	}

	target, source, tokenExceptions, err := buildDiagnosticTargets(c, dc)
	if err != nil {
		fail(err.Error())
	}

	report, err := diagnoseRange(target, source, tokenExceptions, dc.ChainId, *from, *to, *contract)
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fail(err.Error())
	}
	if err != nil {
		fail(err.Error())
	}
}

// buildDiagnosticTargets creates the parser target and raw source reader used by dry-run diagnosis.
func buildDiagnosticTargets(c configs.Config, dc configs.ParserDexConfig) (p_dex.TargetApp, p_dex.SourceDataStore, map[string]bool, error) {
	parserRepo := repo.New(dc.ChainId, c.Rdb)
	tokenExceptions, err := parserRepo.GetTokenExceptions()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load token exceptions: %w", err)
	}

	app, err := dexwiring.NewTargetApp(parserRepo, logging.Discard, dc)
	if err != nil {
		return nil, nil, nil, err
	}

	readStore, err := dexwiring.NewTargetReadStore(c, dc)
	if err != nil {
		return nil, nil, nil, err
	}
	source, err := dexwiring.NewSourceDataStore(dc, c.Rdb, readStore, logging.Discard)
	if err != nil {
		return nil, nil, nil, err
	}

	return app, source, tokenExceptions, nil
}

// diagnoseRange replays parsing for matching raw transactions without writing parser state.
func diagnoseRange(app p_dex.TargetApp, source p_dex.SourceDataStore, tokenExceptions map[string]bool, chainID string, from, to uint64, contract string) (diagnosisReport, error) {
	report := diagnosisReport{
		ChainID:  chainID,
		From:     from,
		To:       to,
		Contract: contract,
		Results:  []diagnosisTxResult{},
	}
	for height := from; height <= to; height++ {
		if err := app.UpdateParsers(tokenExceptions, height); err != nil {
			return report, fmt.Errorf("update parsers at height %d: %w", height, err)
		}
		txs, err := source.GetSourceTxs(height)
		if err != nil {
			return report, fmt.Errorf("get source txs at height %d: %w", height, err)
		}
		for _, tx := range txs {
			if !rawTxContainsContract(tx, contract) {
				continue
			}
			result := diagnosisTxResult{
				Height: height,
				Hash:   tx.Hash,
				Sender: tx.Sender,
			}
			parsedTxs, err := app.ParseTxs(tx, height)
			if err != nil {
				var partial *p_dex.PartialParseQuarantineError
				if errors.As(err, &partial) {
					result.ParsedTxs = partial.ParsedTxs
					result.ParsedTxCount = len(partial.ParsedTxs)
				}
				result.Error = err.Error()
			} else {
				result.ParsedTxs = parsedTxs
				result.ParsedTxCount = len(parsedTxs)
			}
			report.Results = append(report.Results, result)
		}
		if height == math.MaxUint64 {
			break
		}
	}
	return report, nil
}

// rawTxContainsContract reports whether a raw transaction directly mentions the target contract.
func rawTxContainsContract(tx parser.RawTx, contract string) bool {
	if contract == "" {
		return false
	}
	if strings.Contains(tx.Hash, contract) || strings.Contains(tx.Sender, contract) {
		return true
	}
	for _, log := range tx.LogResults {
		for _, attr := range log.Attributes {
			if strings.Contains(attr.Value, contract) {
				return true
			}
		}
	}
	return false
}

func fail(msg string) {
	_, _ = fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
