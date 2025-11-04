package retry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestShouldRetryGrpcStatus(t *testing.T) {
	// Codes are guaranteed to run sequentially from OK to Unauthenticated
	for i := codes.OK; i <= codes.Unauthenticated; i++ {
		shouldRetry := ShouldRetryGrpcStatus(status.New(i, "test"))
		switch i {
		case codes.DeadlineExceeded, codes.ResourceExhausted, codes.Unavailable:
			assert.Truef(t, shouldRetry, "%s should be retried", i)
		default:
			assert.Falsef(t, shouldRetry, "%s should not be retried", i)
		}
	}
}
