package grpc

import (
	"fmt"
	"runtime/debug"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
)

func panicHandler(p interface{}) (err error) {
	if r := recover(); r == nil {
		err = fmt.Errorf("%v", p)
		log.Errorf("Caught panic in gRPC call. Stack: %s", string(debug.Stack()))
	}
	return
}

func (a *apiImpl) recoveryOpts() []grpc_recovery.Option {
	return []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(panicHandler),
	}
}

func (a *apiImpl) unaryRecovery() grpc.UnaryServerInterceptor {
	return grpc_recovery.UnaryServerInterceptor(a.recoveryOpts()...)
}

func (a *apiImpl) streamRecovery() grpc.StreamServerInterceptor {
	return grpc_recovery.StreamServerInterceptor(a.recoveryOpts()...)
}
