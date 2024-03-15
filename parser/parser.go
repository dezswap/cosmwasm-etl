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
	MatchedToParsedTx(eventlog.MatchedResult, ...interface{}) (*T, error)
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
	dtos := []*T{}

	for _, match := range matched {
		dto, err := p.MatchedToParsedTx(match, optionals...)
		if err != nil {
			return nil, errors.Wrap(err, "parserImpl.Parse")
		}
		// skip if no dto
		if dto == nil {
			continue
		}

		overridden, err := defaultVal.Override(*dto)
		if err != nil {
			return nil, errors.Wrap(err, "parserImpl.Parse")
		}
		dtos = append(dtos, &overridden)
	}

	return dtos, nil
}

// matchedToParsedTxDto implements parser
func (p *parserImpl[T]) MatchedToParsedTx(matched eventlog.MatchedResult, optionals ...interface{}) (*T, error) {
	return p.Mapper.MatchedToParsedTx(matched, optionals...)
}
