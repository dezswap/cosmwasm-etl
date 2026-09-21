package dex

import (
	"testing"

	"github.com/dezswap/cosmwasm-etl/parser"
	pdex "github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PartialQuarantineRecorder_Record(t *testing.T) {
	rawTx := parser.RawTx{Hash: "tx"}
	createPairTx := parser.RawTx{
		Hash: "create-pair",
		LogResults: eventlog.LogResults{{
			Type: eventlog.WasmType,
			Attributes: eventlog.Attributes{
				{Key: "action", Value: string(CreatePair)},
			},
		}},
	}

	tcs := []struct {
		desc     string
		tx       parser.RawTx
		err      error
		recorded bool
	}{
		{
			desc:     "ambiguous event",
			tx:       rawTx,
			err:      &eventlog.AmbiguousEventError{Contract: "token", Action: "transfer"},
			recorded: true,
		},
		{
			desc:     "empty event value",
			tx:       rawTx,
			err:      errors.Wrap(pdex.ErrEmptyEventValue, "conx.ParseTxs transfer"),
			recorded: true,
		},
		{
			desc:     "any other failure stays fatal",
			tx:       rawTx,
			err:      errors.New("wrong asset"),
			recorded: false,
		},
		{
			desc:     "create_pair tx stays fatal",
			tx:       createPairTx,
			err:      &eventlog.AmbiguousEventError{Contract: "token", Action: "transfer"},
			recorded: false,
		},
	}

	for _, tc := range tcs {
		recorder := NewPartialQuarantineRecorder(tc.tx, 10)
		require.Equal(t, tc.recorded, recorder.Record("transfer", tc.err), tc.desc)

		err := recorder.Err([]ParsedTx{{Hash: tc.tx.Hash}})
		if !tc.recorded {
			assert.NoError(t, err, tc.desc)
			continue
		}

		var partial *PartialParseQuarantineError
		require.ErrorAs(t, err, &partial, tc.desc)
		assert.Equal(t, PartialQuarantineStagePrefix+"transfer", partial.Quarantine.Stage, tc.desc)
		assert.Equal(t, tc.tx.Hash, partial.Quarantine.Hash, tc.desc)
		assert.Equal(t, uint64(10), partial.Quarantine.Height, tc.desc)
		assert.Equal(t, []ParsedTx{{Hash: tc.tx.Hash}}, partial.ParsedTxs, tc.desc)
	}
}

// Only the first failure is kept: it is the one that made the tx incomplete.
func Test_PartialQuarantineRecorder_KeepsFirstFailure(t *testing.T) {
	recorder := NewPartialQuarantineRecorder(parser.RawTx{Hash: "tx"}, 10)

	require.True(t, recorder.Record("wasm_transfer", &eventlog.AmbiguousEventError{Contract: "first"}))
	require.True(t, recorder.Record("transfer", errors.Wrap(pdex.ErrEmptyEventValue, "second")))

	var partial *PartialParseQuarantineError
	require.ErrorAs(t, recorder.Err(nil), &partial)
	assert.Equal(t, PartialQuarantineStagePrefix+"wasm_transfer", partial.Quarantine.Stage)
	assert.Equal(t, "first", partial.Quarantine.Contract)
}
