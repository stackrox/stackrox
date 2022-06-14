package grpc

import (
	"runtime/debug"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
)

func anyToError(x interface{}) error {
	if x == nil {
		return errors.New("nil panic reason")
	}
	if err, ok := x.(error); ok {
		return err
	}
	return errors.Errorf("%v", x)
}

func panicHandler(p interface{}) error {
	err := anyToError(p)
	utils.Should(errors.Errorf("Caught panic in gRPC call. Reason: %v. Stack trace:\n%s", err, string(debug.Stack())))
	return err
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
