package nodeerr

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const maxBodySnippetLen = 256

// MaxErrorBodyBytes bounds a response the client will not decode: enough to keep a
// grpc-gateway error or a proxy page readable without buffering whatever a broken
// gateway decides to stream. A response the client does decode is the payload
// itself, so how large it may be is the node's call, not this package's.
const MaxErrorBodyBytes = 32 << 10

const (
	TransportRPC  = "rpc"
	TransportLCD  = "lcd"
	TransportGRPC = "grpc"
)

var (
	// ErrHeightUnavailable marks a height the node can never serve; retrying cannot help.
	ErrHeightUnavailable = errors.New("height is not retained by the node")
	// ErrRetryable marks a failure expected to clear on its own.
	ErrRetryable = errors.New("retryable node failure")
)

type Error struct {
	Op        string
	Transport string
	// Host is the node authority only. The full URL stays out of the error because
	// providers put API keys in the path or the query string.
	Host string
	// Height is the block height the request asked for, 0 when it asked for none.
	// It has to be carried explicitly: the request URL that held it is dropped.
	Height uint64
	// Status is the HTTP status, 0 when the failure carried no HTTP response.
	Status int
	// Code is a transport specific code, e.g. a gRPC code name.
	Code string
	// Body is a bounded snippet of the response.
	Body string
	// Detail is a transport specific payload, e.g. a JSON-RPC error object or a
	// gRPC status. Both Detail and Err stay reachable through Unwrap.
	Detail error
	Err    error
	// Class is ErrRetryable, ErrHeightUnavailable, or nil for a permanent failure.
	// Every client records it, but only the rpc client acts on it today: see
	// getWithRetry. For LCD and grpc it is a verdict a future retry loop can read.
	Class error
}

func (e *Error) Error() string {
	msg := e.Op

	// Detail is rendered last because a transport that reports a code already says
	// everything Detail would repeat.
	switch {
	case e.Status != 0:
		msg += ": node returned http " + strconv.Itoa(e.Status)
	case e.Code != "":
		msg += ": node returned " + e.Transport + " " + e.Code
	case e.Detail != nil:
		msg += ": node returned error: " + e.Detail.Error()
	}

	if where := e.where(); where != "" {
		msg += " (" + where + ")"
	}
	if e.Body != "" {
		msg += ": " + e.Body
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

// where names the node and the request. Op alone does not say which node answered
// or which height was asked for, and the URL that held both is deliberately dropped.
func (e *Error) where() string {
	if e.Host == "" && e.Height == 0 {
		return ""
	}

	parts := make([]string, 0, 3)
	if e.Transport != "" {
		parts = append(parts, e.Transport)
	}
	if e.Host != "" {
		parts = append(parts, e.Host)
	}
	if e.Height != 0 {
		parts = append(parts, "height="+strconv.FormatUint(e.Height, 10))
	}
	return strings.Join(parts, " ")
}

func (e *Error) Is(target error) bool { return target == e.Class }

// Unwrap returns every cause. Returning only one of them would leave the other
// rendered in the message but unreachable through errors.Is and errors.As.
func (e *Error) Unwrap() []error {
	switch {
	case e.Detail != nil && e.Err != nil:
		return []error{e.Detail, e.Err}
	case e.Detail != nil:
		return []error{e.Detail}
	case e.Err != nil:
		return []error{e.Err}
	}
	return nil
}

// Retryable reports whether err is worth another attempt.
func Retryable(err error) bool { return errors.Is(err, ErrRetryable) }

// WithHeight stamps the requested height on err and returns it unchanged otherwise,
// so a caller that knows the height can attach it without every construction site
// having to thread it through. Call it before wrapping: fmt.Errorf renders its
// message once, so a height added afterwards reaches errors.As but never the text a
// log line prints.
func WithHeight(err error, height uint64) error {
	var nodeErr *Error
	if errors.As(err, &nodeErr) {
		nodeErr.Height = height
	}
	return err
}

// Transient builds the error for a request that never produced a usable response,
// so it carries no node verdict and stays retryable.
func Transient(op, transport, host string, err error) error {
	return &Error{Op: op, Transport: transport, Host: host, Err: withoutRequestURL(err), Class: ErrRetryable}
}

// withoutRequestURL reduces a *url.Error to its cause. Its own message repeats the
// request URL, which defeats keeping only the host: providers put API keys in the
// path and the query string.
func withoutRequestURL(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return urlErr.Err
	}
	return err
}

// HTTPStatus builds the error for a node response the caller could not use because
// of its status code.
func HTTPStatus(op, transport, host string, status int, body []byte) error {
	return &Error{
		Op:        op,
		Transport: transport,
		Host:      host,
		Status:    status,
		Body:      Snippet(body),
		Class:     classFromHTTPStatus(status),
	}
}

// classFromHTTPStatus treats only overload and server side faults as transient; a
// 4xx is the node refusing the request and will refuse it again.
func classFromHTTPStatus(status int) error {
	if status >= http.StatusInternalServerError || status == http.StatusTooManyRequests {
		return ErrRetryable
	}
	return nil
}

// Snippet bounds a response body so an HTML error page or a truncated JSON payload
// stays readable in a log line without carrying the whole response.
func Snippet(body []byte) string {
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > maxBodySnippetLen {
		return strings.ToValidUTF8(snippet[:maxBodySnippetLen], "") + "..."
	}
	return strings.ToValidUTF8(snippet, "")
}

// HostOf keeps the authority of rawURL and drops everything that can carry a
// credential. An unparsable URL yields "" rather than leaking the raw string.
func HostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}

// RedactedGRPCAddress rebuilds a gRPC target without its userinfo, query and fragment.
// The path stays: it is what names the node in unix:///var/run/node.sock. HostOf cannot
// stand in here, since url.Parse reads a bare host:port as a scheme.
func RedactedGRPCAddress(target string) string {
	u, err := url.Parse(target)
	if err != nil {
		return ""
	}
	u.User = nil
	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""
	u.RawFragment = ""
	return u.String()
}
