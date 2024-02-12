package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
)

func TestGetCentralExpiration(t *testing.T) {
	ctx := context.Background()

	cases := []string{"", time.Now().Format(time.RFC3339), "not a time"}

	for _, timestamp := range cases {
		t.Run(fmt.Sprintf("expired-at %s", timestamp), func(tt *testing.T) {
			tt.Setenv("ROX_EXPIRED_AT", timestamp)

			value, parseErr := time.Parse(time.RFC3339, timestamp)
			var expected *types.Timestamp
			if parseErr == nil {
				expected, _ = types.TimestampProto(value)
			}

			metadata, err := (&serviceImpl{}).GetMetadata(ctx, nil)
			assert.NoError(tt, err, "whatever, no error")
			assert.Equal(tt, expected, metadata.ExpiredAt)
		})
	}
}
