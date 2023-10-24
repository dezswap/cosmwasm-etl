package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ToEpoch(t *testing.T) {
	expected := float64(1666643140)
	actual := ToEpoch(time.Date(2022, 10, 24, 20, 25, 40, 0, time.UTC))

	assert.Equal(t, expected, actual)
}

func Test_ToTime(t *testing.T) {
	expected := time.Date(2022, 10, 24, 20, 25, 40, 0, time.UTC)
	actual := ToTime(float64(1666643140))

	assert.Equal(t, expected, actual)
}
