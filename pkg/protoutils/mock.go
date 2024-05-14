package protoutils

import (
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func GetEqMessageGoMockMatcher(expectedMsg proto.Message) gomock.Matcher {
	return gomock.Cond(func(msg any) bool {
		return proto.Equal(expectedMsg, msg.(proto.Message))
	})
}
