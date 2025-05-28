package service

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestPing(t *testing.T) {
	service := &serviceImpl{}
	response, err := service.Ping(sac.WithNoAccess(context.Background()), &v1.Empty{})
	assert.NoError(t, err)
	protoassert.Equal(t, &v1.PongMessage{Status: "ok"}, response)
}
