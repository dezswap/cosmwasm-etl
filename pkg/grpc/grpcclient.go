package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/stretchr/testify/mock"
)

// BACK_OFF_MAX_DELAY is default max delay for exponential backoff algorithm
const BACK_OFF_MAX_DELAY = 3 * time.Second

var serviceDomain string

type ServiceDesc interface {
	GetConnection(opts ...grpc.DialOption) *grpc.ClientConn
	GetConnectionWithContext(ctx context.Context, opts ...grpc.DialOption) *grpc.ClientConn
	CloseConnection() error
}

func SetServiceDomain(domain string) {
	serviceDomain = domain
}

type serviceDescImpl struct {
	alias           string
	destination     string
	destinationPort int
	backoffMaxDelay time.Duration
	noTLS           bool

	serviceConn *grpc.ClientConn
}

func GetServiceDesc(alias string, c configs.GrpcConfig) ServiceDesc {

	return &serviceDescImpl{
		alias:           alias,
		destination:     c.Host,
		destinationPort: c.Port,
		backoffMaxDelay: c.BackoffDelay.Duration,
		noTLS:           c.NoTLS,
	}
}

// GetConnection returns gRPC client connection
func (c *serviceDescImpl) GetConnection(opts ...grpc.DialOption) *grpc.ClientConn {
	return c.GetConnectionWithContext(context.Background(), opts...)
}

// GetConnectionWithContext returns gRPC client connection with context
func (c *serviceDescImpl) GetConnectionWithContext(ctx context.Context, opts ...grpc.DialOption) *grpc.ClientConn {
	if c.serviceConn != nil {
		return c.serviceConn
	}

	dest := c.destination
	// Alias to name
	if dest != "" && !strings.Contains(dest, ".") && dest != "localhost" {
		if serviceDomain == "" {
			panic("service domain is not set")
		}
		dest = strings.Join([]string{dest, serviceDomain}, ".")
	}
	// Assemble port
	dest = strings.Join([]string{dest, strconv.Itoa(c.destinationPort)}, ":")

	options := opts
	if !c.noTLS {
		// FIXME: it is very slow procedure, we should have cert in local filesystem or cache it
		conn, err := tls.Dial("tcp", dest, &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			logger.Warn("cannot dial to TLS enabled server:", err)
			return nil
		}
		certs := conn.ConnectionState().PeerCertificates
		err = conn.Close()
		if err != nil {
			logger.Warn("cannot close TLS connection:", err)
		}
		pool := x509.NewCertPool()
		pool.AddCert(certs[0])

		clientCert := credentials.NewClientTLSFromCert(pool, "")
		options = append(options, grpc.WithTransportCredentials(clientCert))
	} else {
		options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	backoff := backoff.DefaultConfig
	// We apply backoff max delay (3 seconds by default)
	backoff.MaxDelay = func() time.Duration {
		delay := c.backoffMaxDelay
		if delay != 0 {
			return delay
		}
		return BACK_OFF_MAX_DELAY
	}()

	options = append(options, grpc.WithConnectParams(grpc.ConnectParams{
		Backoff: backoff,
	}))

	logger.Info("dial [", c.alias, "]: ", dest)
	conn, err := grpc.NewClient(dest, options...)
	if err != nil {
		// It is very likely unreachable code under non-blocking dialing
		logger.Panic("cannot connect to gRPC server:", err)
	}
	c.serviceConn = conn

	// State change logger
	go func() {
		isReady := false

		for {
			s := conn.GetState()
			if s == connectivity.Shutdown {
				logger.Info("connection state [", c.alias, "]: ", s)
				break
			} else if isReady && s == connectivity.TransientFailure {
				logger.Info("connection state [", c.alias, "]: ", s)
				isReady = false
			}

			if !conn.WaitForStateChange(ctx, s) {
				// Logging last state just after ctx expired
				// Even this can miss last "shutdown" state. very unlikely.
				last := conn.GetState()
				if s != last {
					logger.Info("connection state [", c.alias, "]: ", last)
				}
				break
			}
		}
	}()

	return c.serviceConn
}

// CloseConnection closes existing connection
func (c *serviceDescImpl) CloseConnection() error {
	if c.serviceConn == nil {
		// Very unlikely
		logger.Warn("connection is already closed or not connected yet")
		return nil
	}

	err := c.serviceConn.Close()
	if err != nil {
		return err
	}

	c.serviceConn = nil
	return nil
}

type ServiceDescMock struct {
	mock.Mock
}

var _ ServiceDesc = &ServiceDescMock{}

func (s *ServiceDescMock) GetConnection(opts ...grpc.DialOption) *grpc.ClientConn {
	return s.Mock.MethodCalled("GetConnection", opts).Get(0).(*grpc.ClientConn)

}
func (s *ServiceDescMock) GetConnectionWithContext(ctx context.Context, opts ...grpc.DialOption) *grpc.ClientConn {
	return s.Mock.MethodCalled("GetConnectionWithContext", ctx, opts).Get(0).(*grpc.ClientConn)

}
func (s *ServiceDescMock) CloseConnection() error {
	return s.Mock.MethodCalled("CloseConnection").Error(0)
}
