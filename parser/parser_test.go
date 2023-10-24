package parser

import (
	"fmt"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type logfinderMock struct{ mock.Mock }
type mapperMock struct{ mock.Mock }

func Test_parser(t *testing.T) {
	type testCase struct {
		expected    []*ParsedTx
		matched     eventlog.MatchedResults
		dto         *ParsedTx
		mapperError string
	}

	var logFinder logfinderMock
	var mapper mapperMock
	setUp := func(t testCase) {
		logFinder = logfinderMock{}
		mapper = mapperMock{}
		logFinder.On("FindFromLogs", mock.Anything).Return(t.matched)

		if t.mapperError == "" {
			mapper.On("matchedToParsedTx", mock.Anything, mock.Anything).Return(t.dto, nil)
		} else {
			mapper.On("matchedToParsedTx", mock.Anything, mock.Anything).Return(&ParsedTx{}, errors.New(t.mapperError))
		}
	}
	parsedTx := &ParsedTx{"hash", time.Time{}, Provide, "sender", "ContractAddr", []Asset{{"Asset0", "100"}, {"Asset1", "100"}}, "Lp", "1000", "", make(map[string]interface{})}

	tcs := []testCase{
		{[]*ParsedTx{}, eventlog.MatchedResults{}, &ParsedTx{}, ""},
		{nil, eventlog.MatchedResults{eventlog.MatchedResult{{Key: "key", Value: "value"}}}, &ParsedTx{}, "mapper errors"},
		{
			[]*ParsedTx{parsedTx, parsedTx},
			eventlog.MatchedResults{
				[]eventlog.MatchedItem{{Key: "Key", Value: "First"}},
				[]eventlog.MatchedItem{{Key: "Key", Value: "Second"}},
			},
			parsedTx,
			"",
		},
	}

	for idx, tc := range tcs {
		setUp(tc)
		msg := fmt.Sprintf("tcs(%d): ", idx)
		assert := assert.New(t)
		parser := NewParser(&logFinder, &mapper)

		dtos, err := parser.Parse("hash", time.Time{}, eventlog.LogResults{})

		assert.Equal(tc.expected, dtos, fmt.Sprintf("%s must return expected dtos", msg))

		if tc.mapperError != "" {
			assert.Error(err, fmt.Sprintf("%s must return error, err(%s)", msg, err))
		}

	}
}

var _ eventlog.LogFinder = &logfinderMock{}
var _ Mapper = &mapperMock{}

// matchedToParsedTx implements mapper
func (m *mapperMock) MatchedToParsedTx(result eventlog.MatchedResult, optionals ...interface{}) (*ParsedTx, error) {
	args := m.Mock.MethodCalled("matchedToParsedTx", result, optionals)
	return args.Get(0).(*ParsedTx), args.Error(1)
}

// FindFromAttrs implements eventlog.LogFinder
func (m *logfinderMock) FindFromAttrs(attrs eventlog.Attributes) eventlog.MatchedResults {
	return m.Mock.MethodCalled("FindFromAttrs", attrs).Get(0).(eventlog.MatchedResults)
}

// FindFromLogs implements eventlog.LogFinder
func (m *logfinderMock) FindFromLogs(logs eventlog.LogResults) eventlog.MatchedResults {
	return m.Mock.MethodCalled("FindFromLogs", logs).Get(0).(eventlog.MatchedResults)
}
