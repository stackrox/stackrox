package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/mitre/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
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

	srv := New(common.Singleton())
	resp, err := srv.ListMitreAttackVectors(context.Background(), &v1.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, []*storage.MitreAttackVector{
		common.MitreTestData["TA0006"],
		common.MitreTestData["TA0005"],
	}, resp.GetMitreAttackVectors())
}
