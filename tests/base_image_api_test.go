//go:build test_e2e || test_compatibility

package tests

import (
	"context"
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
)

func TestBaseImageServicePing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewBaseImageServiceV2Client(conn)

	cases := []struct {
		name string
		call func() error
	}{
		{
			name: "GetBaseImageReferences",
			call: func() error {
				_, err := client.GetBaseImageReferences(ctx, &v2.Empty{})
				return err
			},
		},
		{
			name: "CreateBaseImageReference",
			call: func() error {
				// Passing empty request; we expect a validation error, not 'Unimplemented'
				_, err := client.CreateBaseImageReference(ctx, &v2.CreateBaseImageReferenceRequest{})
				return err
			},
		},
		{
			name: "UpdateBaseImageTagPattern",
			call: func() error {
				_, err := client.UpdateBaseImageTagPattern(ctx, &v2.UpdateBaseImageTagPatternRequest{})
				return err
			},
		},
		{
			name: "DeleteBaseImageReference",
			call: func() error {
				_, err := client.DeleteBaseImageReference(ctx, &v2.DeleteBaseImageReferenceRequest{})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()

			// If the service wasn't registered, it would likely return Unimplemented.
			if err != nil {
				st, ok := status.FromError(err)
				assert.True(t, ok, "Expected a gRPC status error")
				assert.NotEqual(t, codes.Unimplemented, st.Code(), "Endpoint %s is not implemented on the server", tc.name)
			}
		})
	}
}
