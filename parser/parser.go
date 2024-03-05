package parser

import (
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/pkg/errors"
)

type Overrider[T any] interface {
	Override(new T) (T, error)
}
type Mapper[T any] interface {
	// return nil if the matched result is not for this parser
	MatchedToParsedTx(eventlog.MatchedResult, ...interface{}) ([]*T, error)
}

type Parser[T any] interface {
	Parse(raws eventlog.LogResults, defaultVal Overrider[T], optionals ...interface{}) ([]*T, error)
	Mapper[T]
}

type parserImpl[T any] struct {
	Mapper[T]
	eventlog.LogFinder
}

func NewParser[T any](finder eventlog.LogFinder, mapper Mapper[T]) Parser[T] {
	return &parserImpl[T]{mapper, finder}
}

// parse implements parser
func (p *parserImpl[T]) Parse(raws eventlog.LogResults, defaultVal Overrider[T], optionals ...interface{}) ([]*T, error) {
	matched := p.FindFromLogs(raws)
	txs := []*T{}

	for _, match := range matched {
		dtos, err := p.MatchedToParsedTx(match, optionals...)
		if err != nil {
			return nil, errors.Wrap(err, "parserImpl.Parse")
		}
		// skip if no dto
		if dtos == nil {
			continue
		}

		for _, d := range dtos {
			overridden, err := defaultVal.Override(*d)
			if err != nil {
				return nil, errors.Wrap(err, "parserImpl.Parse")
			}
			txs = append(txs, &overridden)
		}
	}

	return txs, nil
}

// matchedToParsedTxDto implements parser
func (p *parserImpl[T]) MatchedToParsedTx(matched eventlog.MatchedResult, optionals ...interface{}) ([]*T, error) {
	return p.Mapper.MatchedToParsedTx(matched, optionals...)
}
