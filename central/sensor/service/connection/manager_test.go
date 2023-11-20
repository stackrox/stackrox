package connection

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_AvoidLargePayloads(t *testing.T) {
	wrappedErr := errors.Wrap(status.Error(codes.ResourceExhausted, "gRPC exhausted"), "recv error")
	errToPreferenece := map[error]bool{
		wrappedErr: false,
		status.Error(codes.ResourceExhausted, "gRPC exhausted"): false,
		status.Error(codes.Canceled, "gRPC canceled"):           true,
		status.Error(codes.Internal, "gRPC internal"):           true,
		nil:                      true,
		errors.New("custom err"): true,
	}
	for err, pref := range errToPreferenece {
		m := manager{}
		m.handleConnectionError("1234", err)
		assert.Equal(t, pref, m.GetConnectionPreference("1234").SendDeduperState)
	}
}
