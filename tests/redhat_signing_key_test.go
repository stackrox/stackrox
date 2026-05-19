//go:build test_e2e

package tests

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

const (
	redHatIntegrationID = "io.stackrox.signatureintegration.12a37a37-760e-4388-9e79-d62726c075b2"
	watchIntervalEnv    = "ROX_REDHAT_SIGNING_KEY_WATCH_INTERVAL"
	shortWatchInterval  = "10s"

	testPublicKeyPEM1 = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE16IoQbiiB5exTRLTkl2rn5FuyXys
4TbDn4+GhQD1JmLZnAiA0cXktX+gFdxu/0JM9pcjjaqT7pdXztbBs78cXg==
-----END PUBLIC KEY-----
`
	testPublicKeyPEM2 = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEQq1X/6XxCA4s0++8Tvl8k+Z0G/GN
LKpdYJEldXnyRE4ppY5d7vnRZHvdZQMSE3KoRSMvVnzZtc9LTKLB3DlS/w==
-----END PUBLIC KEY-----
`
)

type bundleKey struct {
	Name string `json:"name"`
	PEM  string `json:"pem"`
}

type keyBundle struct {
	Keys []bundleKey `json:"keys"`
}

type RedHatSigningKeySuite struct {
	KubernetesSuite
	conn *grpc.ClientConn
}

func TestRedHatSigningKey(t *testing.T) {
	suite.Run(t, new(RedHatSigningKeySuite))
}

func (s *RedHatSigningKeySuite) SetupSuite() {
	s.KubernetesSuite.SetupSuite()
	s.conn = centralgrpc.GRPCConnectionToCentral(s.T())
}

func (s *RedHatSigningKeySuite) siClient() v1.SignatureIntegrationServiceClient {
	return v1.NewSignatureIntegrationServiceClient(s.conn)
}

func (s *RedHatSigningKeySuite) listIntegrations(ctx context.Context) (*v1.ListSignatureIntegrationsResponse, error) {
	return s.siClient().ListSignatureIntegrations(ctx, &v1.Empty{})
}

// waitForIntegrationKeys polls until the Red Hat integration has exactly the expected key names.
func (s *RedHatSigningKeySuite) waitForIntegrationKeys(ctx context.Context, expectedNames []string, description string) {
	t := s.T()
	sort.Strings(expectedNames)
	mustEventually(t, ctx, func() error {
		rpcCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		resp, err := s.listIntegrations(rpcCtx)
		if err != nil {
			return fmt.Errorf("listing integrations: %w", err)
		}
		for _, si := range resp.GetIntegrations() {
			if si.GetId() != redHatIntegrationID {
				continue
			}
			keys := si.GetCosign().GetPublicKeys()
			if len(keys) != len(expectedNames) {
				return fmt.Errorf("expected %d keys, got %d", len(expectedNames), len(keys))
			}
			names := make([]string, len(keys))
			for i, k := range keys {
				names[i] = k.GetName()
			}
			sort.Strings(names)
			for i := range expectedNames {
				if names[i] != expectedNames[i] {
					return fmt.Errorf("expected key names %v, got %v", expectedNames, names)
				}
			}
			return nil
		}
		return fmt.Errorf("integration %q not found", redHatIntegrationID)
	}, 5*time.Second, description)
}

func (s *RedHatSigningKeySuite) TestDefaultIntegrationExists() {
	t := s.T()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := s.listIntegrations(ctx)
	s.Require().NoError(err, "listing signature integrations")

	var found bool
	for _, si := range resp.GetIntegrations() {
		if si.GetId() != redHatIntegrationID {
			continue
		}
		found = true
		s.Assert().Equal("Red Hat", si.GetName())
		s.Assert().GreaterOrEqual(len(si.GetCosign().GetPublicKeys()), 1,
			"expected at least one cosign public key")
		for _, pk := range si.GetCosign().GetPublicKeys() {
			s.Assert().NotEmpty(pk.GetName(), "key name must not be empty")
			s.Assert().NotEmpty(pk.GetPublicKeyPemEnc(), "key PEM must not be empty")
		}
		break
	}
	s.Require().True(found, "Red Hat signature integration %q not found", redHatIntegrationID)
	t.Logf("Red Hat integration found with %d key(s)",
		len(resp.GetIntegrations()[0].GetCosign().GetPublicKeys()))
}

func (s *RedHatSigningKeySuite) TestWatcherPicksUpBundleFile() {
	t := s.T()
	ns := namespaces.StackRox
	testCtx, overallCtx, cancel := testContexts(t, "TestWatcherPicksUpBundleFile", 10*time.Minute)
	defer cancel()

	defer func() {
		s.logf("Cleanup: removing %s env var", watchIntervalEnv)
		s.mustDeleteDeploymentEnvVar(overallCtx, ns, "central", watchIntervalEnv)
		s.waitUntilK8sDeploymentReady(overallCtx, ns, "central")
	}()

	s.logf("Setting %s=%s on central", watchIntervalEnv, shortWatchInterval)
	s.mustSetDeploymentEnvVal(testCtx, ns, "central", "central", watchIntervalEnv, shortWatchInterval)
	s.waitUntilK8sDeploymentReady(testCtx, ns, "central")

	bundle := keyBundle{
		Keys: []bundleKey{
			{Name: "test-key-1", PEM: testPublicKeyPEM1},
			{Name: "test-key-2", PEM: testPublicKeyPEM2},
		},
	}
	bundleJSON, err := json.Marshal(bundle)
	s.Require().NoError(err)

	b64 := base64.StdEncoding.EncodeToString(bundleJSON)
	writeCmd := fmt.Sprintf("mkdir -p /tmp/redhat-signing-keys && echo %s | base64 -d > /tmp/redhat-signing-keys/bundle.json", b64)

	s.logf("Writing test key bundle to Central pod")
	execInDeployment(t, s.k8s, "central", ns, "sh", "-c", writeCmd)

	defer func() {
		s.logf("Cleanup: removing test bundle file")
		execInDeployment(t, s.k8s, "central", ns, "sh", "-c", "rm -f /tmp/redhat-signing-keys/bundle.json")
	}()

	s.logf("Waiting for watcher to pick up the bundle")
	s.waitForIntegrationKeys(testCtx, []string{"test-key-1", "test-key-2"},
		"watcher did not pick up the bundle file")

	t.Log("Watcher successfully picked up the bundle with 2 test keys")
}
