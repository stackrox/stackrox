package audit

import (
	"context"

	"google.golang.org/grpc"
)

// Auditor implements a unary server interceptor
type Auditor interface {
	UnaryServerInterceptor() func(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error)
}
