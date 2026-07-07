package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	collectorrepo "github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	pds "github.com/dezswap/cosmwasm-etl/parser/dex/dezswap"
	"github.com/dezswap/cosmwasm-etl/parser/dex/repo"
	"github.com/dezswap/cosmwasm-etl/parser/dex/srcstore"
	ts_srcstore "github.com/dezswap/cosmwasm-etl/parser/dex/srcstore/terraswap"
	psf "github.com/dezswap/cosmwasm-etl/parser/dex/starfleit"
	pts "github.com/dezswap/cosmwasm-etl/parser/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/httpclient"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/s3client"
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
	if err != nil {
		fail(err.Error())
	}
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
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

	var app p_dex.TargetApp
	switch dc.TargetApp {
	case dex.Terraswap:
		app, err = pts.New(parserRepo, logging.Discard, dc)
	case dex.Dezswap:
		app, err = pds.New(parserRepo, logging.Discard, dc, dc.ChainId)
	case dex.Starfleit:
		app, err = psf.New(parserRepo, logging.Discard, dc, dc.ChainId)
	default:
		return nil, nil, nil, fmt.Errorf("unknown target app: %s", dc.TargetApp)
	}
	if err != nil {
		return nil, nil, nil, err
	}

	var source p_dex.SourceDataStore
	if dc.TargetApp == dex.Terraswap {
		fallback, err := ts_srcstore.NewFromConfig(dc.NodeConfig, dc.FactoryAddress)
		if err != nil {
			return nil, nil, nil, err
		}
		source = srcstore.NewCollectorFallback(dc.ChainId, collectorrepo.New(c.Rdb), fallback, logging.Discard)
	} else {
		source = srcstore.New(getDexCollectorReadStore(c, dc))
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

// getDexCollectorReadStore mirrors the production parser source selection for collector-backed DEXes.
func getDexCollectorReadStore(c configs.Config, dc configs.ParserDexConfig) datastore.ReadStore {
	nodeConf := dc.NodeConfig
	if nodeConf.GrpcConfig.Host != "" {
		serviceDesc := grpc.GetServiceDesc("collector", nodeConf.GrpcConfig)

		store, err := datastore.New(c, serviceDesc, nil)
		if err != nil {
			panic(err)
		}
		if nodeConf.FailoverLcdHost != "" {
			failoverStore, err := datastore.New(
				c,
				serviceDesc,
				datastore.NewLcdClient(nodeConf.FailoverLcdHost, httpclient.New(dc.NodeConfig.HttpClientConfig)),
			)
			if err != nil {
				panic(err)
			}
			store = failoverStore
		}

		return datastore.NewReadStoreWithGrpc(dc.ChainId, store)
	}

	s3Client, err := s3client.NewClient(c.S3)
	if err != nil {
		panic(err)
	}
	return datastore.NewReadStore(dc.ChainId, s3Client)
}

func fail(msg string) {
	_, _ = fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
