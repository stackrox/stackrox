package protomock

import (
	"github.com/stackrox/rox/pkg/protocompat"
	"go.uber.org/mock/gomock"
)

func GoMockMatcherEqualMessage[T protocompat.Equalable[T]](expectedMsg T) gomock.Matcher {
	return gomock.Cond(func(msg any) bool {
		return protocompat.Equal(expectedMsg, toT(expectedMsg, msg))
	})
}

func toT[T any](_ T, a any) T {
	return a.(T)
}
