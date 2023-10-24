package router

import (
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/stretchr/testify/mock"
)

type routerMock struct {
	mock.Mock
}

var _ Router = &routerMock{}

func NewRouterMock() Router {
	return &routerMock{}
}

// Run implements Router
func (*routerMock) Run() {}

// Logger implements Router
func (*routerMock) Logger() logging.Logger {
	return nil
}

// RouterAddress implements Router
func (r *routerMock) RouterAddress() string {
	args := r.Mock.MethodCalled("RouterAddress")
	return args.Get(0).(string)
}

// Routes implements Router
func (r *routerMock) Routes(from string, to string) [][]string {
	args := r.Mock.MethodCalled("Routes", from, to)
	return args.Get(0).([][]string)
}

// TokensFrom implements Router
func (r *routerMock) TokensFrom(from string, hopCount int) []string {
	args := r.Mock.MethodCalled("TokensFrom", from, hopCount)
	return args.Get(0).([]string)
}

// Update implements Router
func (r *routerMock) Update() error {
	args := r.Mock.MethodCalled("Update")
	return args.Error(0)
}
