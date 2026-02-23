package datastore

import "github.com/pkg/errors"

var (
	// ErrGrpcUnknownCode is returned when a gRPC call responds with codes.Unknown.
	// See dataStoreImpl.skipUnknownCode for when this error is intentionally skipped.
	ErrGrpcUnknownCode = errors.New("gRPC unknown status code")
)
