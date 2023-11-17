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
		wrappedErr: true,
		status.Error(codes.ResourceExhausted, "gRPC exhausted"): true,
		status.Error(codes.Canceled, "gRPC canceled"):           false,
		status.Error(codes.Internal, "gRPC internal"):           false,
		nil:                      false,
		errors.New("custom err"): false,
	}
	for err, pref := range errToPreferenece {
		m := manager{}
		m.handleConnectionError("1234", err)
		assert.Equal(t, pref, m.GetConnectionPreference("1234").AvoidLargeSyncPayloads)
	}
}
