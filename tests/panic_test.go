//go:build test_e2e

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestPanicSuite(t *testing.T) {
	suite.Run(t, new(PanicSuite))
}

func TestHidePanicSuite(t *testing.T) {
	suite.Run(t, new(HidePanicSuite))
}

type PanicSuite struct {
	KubernetesSuite
}

func (s *PanicSuite) TestPanic() {
	t := s.T()

	if isOCP() {
		t.Skip("Skipping because on OCP")
	}

	causeCentralPanic(t)

	// Fake work.
	time.Sleep(30 * time.Second)
}

type HidePanicSuite struct {
	KubernetesSuite
}

func (s *HidePanicSuite) TestHidePanic() {
	t := s.T()

	if !isOCP() {
		t.Skip("Skipping because NOT on OCP")
	}

	causeCentralPanic(t)

	// Fake work.
	time.Sleep(30 * time.Second)

	// Set an env var to trigger a restart.
	ctx := context.Background()
	s.setDeploymentEnvVal(ctx, "stackrox", "central", "central", "NOW", fmt.Sprintf("%s", time.Now().UTC()))
}

func isOCP() bool {
	return os.Getenv("ORCHESTRATOR_FLAVOR") == "openshift"
}

func causeCentralPanic(t *testing.T) {
	ctx := context.Background()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewDelegatedRegistryConfigServiceClient(conn)

	resp, err := service.UpdateConfig(ctx, &v1.DelegatedRegistryConfig{
		EnabledFor:       v1.DelegatedRegistryConfig_ALL,
		DefaultClusterId: "DAVEPANIC", // trigger a central panic on purpose
	})
	require.NoError(t, err)

	dataB, err := json.Marshal(resp)
	require.NoError(t, err)
	t.Logf("Resp: %s", string(dataB))
}
