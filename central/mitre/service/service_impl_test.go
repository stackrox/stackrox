package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/mitre/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
)

func TestMitreAttack(t *testing.T) {
	envIso := envisolator.NewEnvIsolator(t)
	envIso.Setenv(features.SystemPolicyMitreFramework.EnvVar(), "true")
	defer envIso.RestoreAll()

	if !features.SystemPolicyMitreFramework.Enabled() {
		t.Skip("RHACS System Policy MITRE ATT&CK framework feature is disabled. skipping...")
	}

	srv := New(datastore.Singleton())
	resp, err := srv.ListMitreAttackVectors(context.Background(), &v1.Empty{})
	assert.NoError(t, err)
	assert.True(t, len(resp.GetMitreAttackVectors()) > 0)
}

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}
