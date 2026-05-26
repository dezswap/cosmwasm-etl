package schemas

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

func (CollectorBlock) TableName() string {
	return "collector_blocks"
}

func (CollectorPoolSnapshot) TableName() string {
	return "collector_pool_snapshots"
}

func (CollectorSyncedHeight) TableName() string {
	return "collector_synced_heights"
}

func (CollectorJSON) GormDataType() string {
	return "json"
}

func (j *CollectorJSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("failed to unmarshal JSONB value:", value))
	}

	*j = append((*j)[0:0], bytes...)
	return nil
}

func (j CollectorJSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}
