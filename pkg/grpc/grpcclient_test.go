package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/dezswap/cosmwasm-etl/pkg/grpc/baseservice/generated/baseservice"
	"github.com/stretchr/testify/assert"
)

var testFinished = make(chan bool)

type mockService struct {
	baseservice.UnimplementedBaseServer
}

// Predefined `TokenRPC` service in protobuf definition
func (s *mockService) GetServiceInfo(ctx context.Context, in *emptypb.Empty) (*baseservice.ServiceInfoResponse, error) {
	return &baseservice.ServiceInfoResponse{}, nil
}

func initMockGrpcServer() {
	service := &mockService{}
	server := grpc.NewServer()

	baseservice.RegisterBaseServer(server, service)

	lis, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}

	go func() {
		if err := server.Serve(lis); err != nil {
			logger.Debug(err)
		}
	}()

	go func() {
		if <-testFinished {
			server.GracefulStop()
		}
	}()
}

var testServiceDesc = serviceDescImpl{
	destination:     "localhost",
	destinationPort: 9090,
	backoffMaxDelay: time.Second * 60,
	noTLS:           true,
}

func TestGrpcConnection(t *testing.T) {
	go func() {
		initMockGrpcServer()
	}()

	time.Sleep(time.Second)

	conn := testServiceDesc.GetConnection()
	assert.NotNil(t, conn)

	fmt.Println(conn.GetState())

	err := testServiceDesc.CloseConnection()
	assert.Nil(t, err)
}
