package schemas

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectorTableNames(t *testing.T) {
	require.Equal(t, "collector_blocks", CollectorBlock{}.TableName())
	require.Equal(t, "collector_pool_snapshots", CollectorPoolSnapshot{}.TableName())
	require.Equal(t, "collector_synced_heights", CollectorSyncedHeight{}.TableName())
}

func TestCollectorJSONGormDataType(t *testing.T) {
	require.Equal(t, "json", CollectorJSON{}.GormDataType())
}

func TestCollectorJSONScanCopiesBytePayload(t *testing.T) {
	payload := []byte(`{"height":10}`)
	var target CollectorJSON

	err := target.Scan(payload)
	payload[0] = '['

	require.NoError(t, err)
	require.Equal(t, CollectorJSON(`{"height":10}`), target)
}

func TestCollectorJSONScanRejectsUnexpectedValueType(t *testing.T) {
	var target CollectorJSON

	err := target.Scan(`{"height":10}`)

	require.ErrorContains(t, err, "failed to unmarshal JSONB value")
}

func TestCollectorJSONValueSerializesEmptyAndNonEmptyPayloads(t *testing.T) {
	empty, err := CollectorJSON(nil).Value()
	require.NoError(t, err)
	require.Equal(t, []byte("null"), empty)

	payload, err := CollectorJSON(`{"height":10}`).Value()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"height":10}`), payload)
}
