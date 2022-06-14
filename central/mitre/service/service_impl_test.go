package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/mitre/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stretchr/testify/assert"
)

func TestMitreAttack(t *testing.T) {
	srv := New(datastore.Singleton())
	resp, err := srv.ListMitreAttackVectors(context.Background(), &v1.Empty{})
	assert.NoError(t, err)
	assert.True(t, len(resp.GetMitreAttackVectors()) > 0)
}

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}
