package protomock

import (
	"github.com/stackrox/rox/pkg/protocompat"
	"go.uber.org/mock/gomock"
)

func GoMockMatcherEqualMessage(expectedMsg protocompat.Message) gomock.Matcher {
	return gomock.Cond(func(msg any) bool {
		return protocompat.Equal(expectedMsg, msg.(protocompat.Message))
	})
}
