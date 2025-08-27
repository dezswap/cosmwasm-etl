package util

const DefaultErrorDataLength = 512

func TruncateBytes(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "...(truncated)"
	}
	return string(b)
}
