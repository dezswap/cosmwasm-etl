package util

import "time"

func ToEpoch(t time.Time) float64 {
	return float64(t.UnixMicro()) / 1_000_000
}

func ToTime(epoch float64) time.Time {
	return time.UnixMicro(int64(epoch * 1_000_000)).UTC()
}
