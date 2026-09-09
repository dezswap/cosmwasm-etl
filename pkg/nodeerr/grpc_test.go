package nodeerr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_FromGRPC_KeepsTheStatusCode(t *testing.T) {
	err := FromGRPC("op", "grpc.example:443", status.Error(codes.Unavailable, "connection closed"))

	var nodeErr *Error
	require.ErrorAs(t, err, &nodeErr)
	assert.Equal(t, codes.Unavailable.String(), nodeErr.Code)
	assert.Equal(t, "connection closed", nodeErr.Body)
	assert.Equal(t, TransportGRPC, nodeErr.Transport)
	assert.True(t, Retryable(err))
}

func Test_FromGRPC_Classification(t *testing.T) {
	for code, retryable := range map[codes.Code]bool{
		codes.Unavailable:       true,
		codes.DeadlineExceeded:  true,
		codes.ResourceExhausted: true,
		codes.NotFound:          false,
		codes.InvalidArgument:   false,
		codes.PermissionDenied:  false,
	} {
		err := FromGRPC("op", "host", status.Error(code, "boom"))

		assert.Equal(t, retryable, Retryable(err), code.String())
		assert.False(t, errors.Is(err, ErrHeightUnavailable), code.String())
	}
}

func Test_FromGRPC_NonStatusErrorStaysRetryable(t *testing.T) {
	err := FromGRPC("op", "host", errors.New("dial tcp: connection refused"))

	assert.True(t, Retryable(err))
	assert.ErrorContains(t, err, "connection refused")
}

func Test_FromGRPC_NilIsNotAnError(t *testing.T) {
	assert.NoError(t, FromGRPC("op", "host", nil))
}

func Test_FromGRPC_DoesNotLeakACredentialBearingTarget(t *testing.T) {
	for name, target := range map[string]string{
		"userinfo":   "dns://user:SECRET@resolver/host:443",
		"query":      "dns:///resolver:443?apikey=SECRET",
		"fragment":   "dns:///resolver:443#SECRET",
		"unparsable": "://SECRET",
	} {
		t.Run(name, func(t *testing.T) {
			err := FromGRPC("op", target, status.Error(codes.Unavailable, "down"))

			assert.NotContains(t, err.Error(), "SECRET")
		})
	}
}

// Stripping must leave the plain target this repo dials intact.
func Test_FromGRPC_KeepsAPlainTarget(t *testing.T) {
	err := FromGRPC("op", "grpc-fetchhub.fetch.ai:443", status.Error(codes.Unavailable, "down"))

	assert.Contains(t, err.Error(), "grpc-fetchhub.fetch.ai:443")
}
