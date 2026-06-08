package schemas

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type JSON json.RawMessage

func (ParsedTx) TableName() string {
	return "parsed_tx"
}
func (PoolInfo) TableName() string {
	return "pool_info"
}
func (Pair) TableName() string {
	return "pair"
}
func (SyncedHeight) TableName() string {
	return "synced_height"
}
func (PairValidationException) TableName() string {
	return "pair_validation_exception"
}
func (TokenParseException) TableName() string {
	return "token_parse_exception"
}
func (ParseQuarantine) TableName() string {
	return "parse_quarantine"
}

func (Meta) GormDataType() string {
	return "json"
}
func (j *Meta) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	result := Meta{}
	err := json.Unmarshal(bytes, &result)
	*j = Meta(result)
	return err
}

// Value return json value, implement driver.Valuer interface
func (j Meta) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.Marshal(j)
}

func (JSON) GormDataType() string {
	return "json"
}

func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}
	*j = append((*j)[:0], bytes...)
	return nil
}

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}
