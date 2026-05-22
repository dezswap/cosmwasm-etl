package collector

import (
	"errors"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/stretchr/testify/require"
)

type heightCollectorMock struct {
	localHeight  uint64
	localErr     error
	sourceHeight uint64
	sourceErr    error
	collectErr   error
	collected    []uint64
}

func (m *heightCollectorMock) LocalHeight() (uint64, error) {
	return m.localHeight, m.localErr
}

func (m *heightCollectorMock) SourceHeight() (uint64, error) {
	return m.sourceHeight, m.sourceErr
}

func (m *heightCollectorMock) CollectHeight(height uint64) error {
	if m.collectErr != nil {
		return m.collectErr
	}
	m.collected = append(m.collected, height)
	m.localHeight = height
	return nil
}

func TestCollectHeightsCollectsBoundedRange(t *testing.T) {
	collector := &heightCollectorMock{
		localHeight:  3,
		sourceHeight: 10,
	}

	err := collectHeights(collector, heightCollectorConfig{
		StartHeight: 5,
		UntilHeight: 7,
	}, logging.Discard)

	require.NoError(t, err)
	require.Equal(t, []uint64{5, 6, 7}, collector.collected)
}

func TestCollectHeightsStopsWhenUntilHeightAlreadyReached(t *testing.T) {
	collector := &heightCollectorMock{
		localHeight:  7,
		sourceHeight: 10,
	}

	err := collectHeights(collector, heightCollectorConfig{
		StartHeight: 5,
		UntilHeight: 7,
	}, logging.Discard)

	require.NoError(t, err)
	require.Empty(t, collector.collected)
}

func TestCollectHeightsReturnsSourceError(t *testing.T) {
	expected := errors.New("source height failed")
	collector := &heightCollectorMock{
		localHeight: 0,
		sourceErr:   expected,
	}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 1,
	}, logging.Discard)

	require.ErrorIs(t, err, expected)
}

func TestCollectHeightsReturnsLocalError(t *testing.T) {
	expected := errors.New("local height failed")
	collector := &heightCollectorMock{localErr: expected}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 1,
	}, logging.Discard)

	require.ErrorIs(t, err, expected)
}

func TestCollectHeightsReturnsCollectError(t *testing.T) {
	expected := errors.New("collect height failed")
	collector := &heightCollectorMock{
		localHeight:  0,
		sourceHeight: 1,
		collectErr:   expected,
	}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 1,
	}, logging.Discard)

	require.ErrorIs(t, err, expected)
	require.Empty(t, collector.collected)
}

func TestBoundedTargetHeight(t *testing.T) {
	require.Equal(t, uint64(7), boundedTargetHeight(10, 7))
	require.Equal(t, uint64(10), boundedTargetHeight(10, 0))
	require.Equal(t, uint64(5), boundedTargetHeight(5, 7))
}

func TestReachedUntilHeight(t *testing.T) {
	require.True(t, reachedUntilHeight(7, 7))
	require.False(t, reachedUntilHeight(6, 7))
	require.False(t, reachedUntilHeight(7, 0))
}
