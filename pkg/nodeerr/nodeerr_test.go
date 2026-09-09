package nodeerr

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Error_Message(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		contains []string
	}{
		{
			name:     "http status carries the code, the host and the body",
			err:      HTTPStatus("lcdClientImpl.GetTx", TransportLCD, "rest.example", http.StatusNotFound, []byte(`{"code":5,"message":"tx not found"}`)),
			contains: []string{"lcdClientImpl.GetTx", "node returned http 404", "(lcd rest.example)", "tx not found"},
		},
		{
			name:     "height joins the node in the parenthetical",
			err:      &Error{Op: "op", Transport: TransportRPC, Host: "rpc.example", Height: 28940967, Status: 502},
			contains: []string{"node returned http 502", "(rpc rpc.example height=28940967)"},
		},
		{
			name:     "grpc carries the code name and the status message",
			err:      &Error{Op: "op", Transport: TransportGRPC, Host: "grpc.example", Code: "NotFound", Body: "tx not found"},
			contains: []string{"node returned grpc NotFound", "(grpc grpc.example)", "tx not found"},
		},
		{
			name:     "transport failure keeps the cause",
			err:      &Error{Op: "op", Transport: TransportRPC, Host: "rpc.example", Err: errors.New("connection refused")},
			contains: []string{"op", "(rpc rpc.example)", "connection refused"},
		},
		{
			name:     "detail is rendered when the transport reports no code",
			err:      &Error{Op: "op", Transport: TransportRPC, Detail: errors.New("Internal error: pruned")},
			contains: []string{"node returned error: Internal error: pruned"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := tc.err.Error()
			for _, want := range tc.contains {
				assert.Contains(t, msg, want)
			}
		})
	}
}

func Test_Error_Classification(t *testing.T) {
	assert.True(t, Retryable(HTTPStatus("op", TransportLCD, "h", http.StatusBadGateway, nil)))
	assert.True(t, Retryable(HTTPStatus("op", TransportLCD, "h", http.StatusTooManyRequests, nil)))
	assert.False(t, Retryable(HTTPStatus("op", TransportLCD, "h", http.StatusNotFound, nil)))

	// A permanent failure must not match either sentinel.
	permanent := HTTPStatus("op", TransportLCD, "h", http.StatusNotFound, nil)
	assert.False(t, errors.Is(permanent, ErrHeightUnavailable))
	assert.False(t, errors.Is(permanent, ErrRetryable))
}

// Rendering a cause in the message but hiding it from errors.Is would make the two
// disagree, so both Detail and Err have to stay reachable.
func Test_Error_UnwrapsEveryCause(t *testing.T) {
	detail := errors.New("detail")
	cause := errors.New("cause")

	both := &Error{Detail: detail, Err: cause}
	require.ErrorIs(t, both, detail)
	require.ErrorIs(t, both, cause)

	require.ErrorIs(t, &Error{Detail: detail}, detail)
	require.ErrorIs(t, &Error{Err: cause}, cause)
	require.Empty(t, (&Error{}).Unwrap())
}

func Test_Snippet_BoundsTheBody(t *testing.T) {
	snippet := Snippet([]byte(strings.Repeat("a", maxBodySnippetLen*2)))

	assert.Len(t, snippet, maxBodySnippetLen+len("..."))
	assert.True(t, strings.HasSuffix(snippet, "..."))
	assert.Equal(t, "trimmed", Snippet([]byte("  trimmed \n")))
}

// Providers put API keys in the userinfo, the path and the query, so only the
// authority may reach a log line.
func Test_HostOf_KeepsOnlyTheAuthority(t *testing.T) {
	assert.Equal(t, "rest.example:443", HostOf("https://rest.example:443/v1/SECRET?apikey=SECRET"))
	assert.Equal(t, "", HostOf("://not a url"))
}

// A gRPC target is not a URL: the bare host:port form has no scheme, and the path
// names the endpoint rather than a resource.
func Test_RedactedGRPCAddress(t *testing.T) {
	for name, tc := range map[string]struct{ target, want string }{
		"bare host and port":        {"grpc-fetchhub.fetch.ai:443", "grpc-fetchhub.fetch.ai:443"},
		"resolver uri":              {"dns:///grpc-fetchhub.fetch.ai:443", "dns:///grpc-fetchhub.fetch.ai:443"},
		"unix socket":               {"unix:///var/run/node.sock", "unix:///var/run/node.sock"},
		"userinfo dropped":          {"dns://user:SECRET@resolver/host:443", "dns://resolver/host:443"},
		"query dropped":             {"dns:///node.example:443?apikey=SECRET", "dns:///node.example:443"},
		"bare query dropped":        {"node.example:443?apikey=SECRET", "node.example:443"},
		"empty query dropped":       {"node.example:443?", "node.example:443"},
		"fragment dropped":          {"dns:///node.example:443#SECRET", "dns:///node.example:443"},
		"unparsable yields nothing": {"://not a target", ""},
		// The path is the endpoint a resolver dials, so dropping it would leave the
		// error with no node at all.
		"endpoint path kept": {"dns:///node.example:443", "dns:///node.example:443"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, RedactedGRPCAddress(tc.target))
		})
	}
}

// A transport that reports a code already says everything Detail would repeat.
func Test_Error_CodeSuppressesDetail(t *testing.T) {
	err := &Error{
		Op: "op", Transport: TransportGRPC, Host: "h",
		Code: "NotFound", Body: "tx not found",
		Detail: errors.New("rpc error: code = NotFound desc = tx not found"),
	}

	assert.Equal(t, "op: node returned grpc NotFound (grpc h): tx not found", err.Error())
}

// A *url.Error repeats the whole request URL, which defeats keeping only the host.
func Test_Transient_DropsTheRequestUrl(t *testing.T) {
	cause := errors.New("connection refused")
	err := Transient("op", TransportLCD, "rest.example", &url.Error{
		Op:  "Get",
		URL: "https://rest.example/SECRET_KEY/cosmos/tx/v1beta1/txs/ABC",
		Err: cause,
	})

	assert.NotContains(t, err.Error(), "SECRET_KEY")
	assert.Contains(t, err.Error(), "connection refused")
	assert.Contains(t, err.Error(), "rest.example")
	assert.True(t, Retryable(err))
	// The cause still has to be reachable for callers that inspect it.
	assert.ErrorIs(t, err, cause)
}

func Test_Transient_KeepsANonUrlError(t *testing.T) {
	cause := errors.New("boom")

	assert.ErrorIs(t, Transient("op", TransportGRPC, "h", cause), cause)
}

// The request url carries the height and is dropped, so the caller that knows it has
// to put it back.
func Test_WithHeight(t *testing.T) {
	t.Run("names the height in the message", func(t *testing.T) {
		err := WithHeight(Transient("op", TransportRPC, "rpc.example", errors.New("refused")), 100)

		assert.Equal(t, "op (rpc rpc.example height=100): refused", err.Error())
	})

	t.Run("leaves an unrelated error alone", func(t *testing.T) {
		cause := errors.New("not a node error")

		assert.Equal(t, cause, WithHeight(cause, 100))
	})

	t.Run("omits the height when none was requested", func(t *testing.T) {
		err := Transient("op", TransportRPC, "rpc.example", errors.New("refused"))

		assert.Equal(t, "op (rpc rpc.example): refused", err.Error())
	})
}

// fmt.Errorf renders its message once, so a height stamped after a wrap reaches
// errors.As but never the text a log line prints. Both call sites stamp first.
func Test_WithHeight_HasToRunBeforeWrapping(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", Transient("op", TransportRPC, "h", errors.New("refused")))

	WithHeight(wrapped, 100)

	var nodeErr *Error
	require.ErrorAs(t, wrapped, &nodeErr)
	require.Equal(t, uint64(100), nodeErr.Height)
	assert.NotContains(t, wrapped.Error(), "height=100")
}
