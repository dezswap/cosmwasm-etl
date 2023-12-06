package aggregator

import (
	"context"
	"github.com/delight-labs/cosmwasm-etl/pkg/logging"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type counterTask struct {
	counter int
}

func (t *counterTask) Execute(_ time.Time, _ time.Time) error {
	t.counter++

	return nil
}
func (t *counterTask) LastProcessedHeight() uint64 {
	return 0
}
func (t *counterTask) StartTimestamp(startTs time.Time) (time.Time, error) {
	return startTs, nil
}

func TestIntervalSchedule(t *testing.T) {
	assert := assert.New(t)

	task := counterTask{}
	scheduler := intervalScheduler{
		task:     &task,
		interval: 1 * time.Second,
		logger:   logging.Discard,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(tk intervalScheduler) {
		if err := tk.Schedule(ctx); err != nil {
			assert.NoError(err)
		}
	}(scheduler)
	time.Sleep(5 * time.Second)
	cancel()

	assert.Equal(task.counter, 6)
}

func TestPredeterminedTimeSchedule(t *testing.T) {
	assert := assert.New(t)

	task := counterTask{}
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: &task,
		interval:              1 * time.Second,
		startTs:               time.Now().Add(-3 * time.Second),
		logger:                logging.Discard,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(tk predeterminedTimeScheduler) {
		if err := tk.Schedule(ctx); err != nil {
			assert.NoError(err)
		}
	}(scheduler)
	time.Sleep(3 * time.Second)
	cancel()

	assert.Equal(task.counter, 6)
}

func TestTimeframe_0_30(t *testing.T) {
	assert := assert.New(t)

	expectedStart := time.Date(2022, 10, 25, 7, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)

	ts := time.Unix(1666681350, 0).UTC() // 2022-10-25 07:02:30 UTC
	actualStart, actualEnd := timeframe(ts, 30*time.Minute)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}

func TestTimeframe_30_0(t *testing.T) {
	assert := assert.New(t)

	expectedStart := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 8, 0, 0, 0, time.UTC)

	ts := time.Unix(1666684022, 0).UTC() // 2022-10-25 07:47:02 UTC
	actualStart, actualEnd := timeframe(ts, 30*time.Minute)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}

func TestTimeframe_InclusiveStart(t *testing.T) {
	assert := assert.New(t)

	expectedStart := time.Date(2022, 10, 25, 7, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)

	ts := time.Unix(1666681200, 0).UTC() // 2022-10-25 07:00:00 UTC
	actualStart, actualEnd := timeframe(ts, 30*time.Minute)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}

func TestTimeframe_ExclusiveEnd(t *testing.T) {
	assert := assert.New(t)

	expectedStart := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 8, 0, 0, 0, time.UTC)

	ts := time.Unix(1666683000, 0).UTC() // 2022-10-25 07:30:00 UTC
	actualStart, actualEnd := timeframe(ts, 30*time.Minute)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}
