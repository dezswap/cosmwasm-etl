package baseservice

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/dezswap/cosmwasm-etl/pkg/grpc/baseservice/generated/baseservice"
)

type BaseServer struct {
	baseservice.UnimplementedBaseServer
}

func (BaseServer) GetServiceInfo(context.Context, *emptypb.Empty) (*baseservice.ServiceInfoResponse, error) {
	return &baseservice.ServiceInfoResponse{}, nil
}
