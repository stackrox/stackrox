//go:build test_e2e || test_compatibility

package tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

			if err != nil {
				st, ok := status.FromError(err)
				assert.True(t, ok, "Expected a gRPC status error")
				assert.NotEqual(t, codes.Unimplemented, st.Code(), "Endpoint %s is not implemented on the server", tc.name)
			}
		})
	}
}

func TestCreateBaseImageReference_Success(t *testing.T) {
	if !reach("https://quay.io", 3*time.Second) {
		t.Skip("Skipping test: quay.io is currently unreachable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewBaseImageServiceV2Client(conn)

	req := &v2.CreateBaseImageReferenceRequest{
		BaseImageRepoPath:   "quay.io/rh_ee_yli3/alpine",
		BaseImageTagPattern: ".*",
	}

	resp, err := client.CreateBaseImageReference(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.GetBaseImageReference())

	ref := resp.GetBaseImageReference()
	assert.Equal(t, req.GetBaseImageRepoPath(), ref.GetBaseImageRepoPath())
	assert.Equal(t, req.GetBaseImageTagPattern(), ref.GetBaseImageTagPattern())
	assert.NotEmpty(t, ref.GetId())
}

func reach(url string, timeout time.Duration) bool {
	client := http.Client{
		Timeout: timeout,
	}
	// Use HEAD to minimize data transfer
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
