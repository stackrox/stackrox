package service

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetMetadata(t *testing.T) {
	testutils.SetMainVersion(t, "4.11.0-testing")
	svc := &serviceImpl{}

	t.Run("logged in user gets version and compatible sensors", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockID := mockIdentity.NewMockIdentity(mockCtrl)
		ctx := authn.ContextWithIdentity(context.Background(), mockID, t)

		metadata, err := svc.GetMetadata(ctx, &v1.Empty{})
		require.NoError(t, err)
		assert.Equal(t, "4.11.0-testing", metadata.GetVersion())
		assert.NotEmpty(t, metadata.GetCompatibleSensorVersions())
	})

	t.Run("anonymous user gets no version", func(t *testing.T) {
		metadata, err := svc.GetMetadata(context.Background(), &v1.Empty{})
		require.NoError(t, err)
		assert.Empty(t, metadata.GetVersion())
		assert.Empty(t, metadata.GetCompatibleSensorVersions())
	})
}
