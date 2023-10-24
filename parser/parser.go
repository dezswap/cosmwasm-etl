package parser

import (
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/pkg/errors"
)

var _ Parser = &parserImpl{}

type parserImpl struct {
	Mapper
	eventlog.LogFinder
}

func NewParser(finder eventlog.LogFinder, mapper Mapper) Parser {
	return &parserImpl{mapper, finder}
}

// parse implements parser
func (p *parserImpl) Parse(hash string, timestamp time.Time, raws eventlog.LogResults, optionals ...interface{}) ([]*ParsedTx, error) {
	matched := p.FindFromLogs(raws)
	dtos := []*ParsedTx{}

	for _, match := range matched {
		dto, err := p.MatchedToParsedTx(match, optionals...)
		if err != nil {
			return nil, errors.Wrap(err, "parse")
		}
		// skip if no dto
		if dto == nil {
			continue
		}
		dto.Hash = hash
		dto.Timestamp = timestamp
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

// matchedToParsedTxDto implements parser
func (p *parserImpl) MatchedToParsedTx(matched eventlog.MatchedResult, optionals ...interface{}) (*ParsedTx, error) {
	return p.Mapper.MatchedToParsedTx(matched, optionals...)
}
