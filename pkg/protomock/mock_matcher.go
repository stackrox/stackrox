package protomock

import (
	"github.com/stackrox/rox/pkg/protoutils"
	"go.uber.org/mock/gomock"
)

func GoMockMatcherEqualMessage[T protoutils.Equalable[T]](expectedMsg T) gomock.Matcher {
	return gomock.Cond(func(msg any) bool {
		return expectedMsg.EqualVT(msg.(T))
	})
}
