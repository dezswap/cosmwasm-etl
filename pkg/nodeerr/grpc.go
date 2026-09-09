package nodeerr

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FromGRPC turns a gRPC call failure into an *Error so the status code survives as
// a field instead of only as text inside the wrapped message.
func FromGRPC(op, target string, err error) error {
	if err == nil {
		return nil
	}

	host := RedactedGRPCAddress(target)

	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC status, so the call carries no node verdict.
		return Transient(op, TransportGRPC, host, err)
	}

	// The status goes in Detail, not Err: Code and Body already carry what its
	// "rpc error: code = X desc = Y" text would repeat.
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

// classFromGRPCCode retries "busy or degraded" and treats anything the node decided
// about the request as permanent. NotFound is permanent but deliberately not
// ErrHeightUnavailable: a tx the node never indexed is not a height it dropped.
func classFromGRPCCode(c codes.Code) error {
	switch c {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted, codes.Internal, codes.Unknown:
		return ErrRetryable
	default:
		return nil
	}
}
