package nodeerr

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FromGRPC turns a gRPC call failure into an *Error so the status code survives as
// a field instead of only as text inside the wrapped message.
func FromGRPC(op, host string, err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC status, so the call carries no node verdict.
		return Transient(op, TransportGRPC, host, err)
	}

	// The status goes in Detail, not Err: Code and Body already carry everything its
	// "rpc error: code = X desc = Y" text would repeat, and Detail stays unwrappable.
	return &Error{
		Op:        op,
		Transport: TransportGRPC,
		Host:      host,
		Code:      st.Code().String(),
		Body:      Snippet([]byte(st.Message())),
		Detail:    err,
		Class:     classFromGRPCCode(st.Code()),
	}
}

// classFromGRPCCode mirrors the RPC client's verdicts: a code that means "busy or
// degraded" is retryable, and anything the node decided about the request itself is
// permanent. Nothing retries grpc calls yet, so this only labels the error for now.
// NotFound stays permanent but is deliberately not ErrHeightUnavailable, since a tx
// the node never indexed is not the same as a height it dropped.
func classFromGRPCCode(c codes.Code) error {
	switch c {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted, codes.Internal, codes.Unknown:
		return ErrRetryable
	default:
		return nil
	}
}
