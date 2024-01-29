package connection

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_GetConnectionPreference(t *testing.T) {
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
		var testName string
		if err == nil {
			testName = "nil"
		} else {
			testName = err.Error()
		}
		t.Run(testName, func(t *testing.T) {
			m := manager{}
			m.handleConnectionError("1234", err)
			assert.Equal(t, pref, m.GetConnectionPreference("1234").SendDeduperState)
		})
	}
}

func Test_GetConnectionPreference_DefaultsToTrue(t *testing.T) {
	m := manager{}
	assert.Equal(t, true, m.GetConnectionPreference("1234").SendDeduperState)
}
