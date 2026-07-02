//go:build test_e2e

package tests

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	redHatIntegrationID = signatures.DefaultRedHatIntegrationID
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
	slices.Sort(expectedNames)
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
			slices.Sort(names)
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

	var matched *storage.SignatureIntegration
	for _, si := range resp.GetIntegrations() {
		if si.GetId() == redHatIntegrationID {
			matched = si
			break
		}
	}
	s.Require().NotNil(matched, "Red Hat signature integration %q not found", redHatIntegrationID)
	s.Assert().Equal("Red Hat", matched.GetName())
	s.Assert().GreaterOrEqual(len(matched.GetCosign().GetPublicKeys()), 1,
		"expected at least one cosign public key")
	for _, pk := range matched.GetCosign().GetPublicKeys() {
		s.Assert().NotEmpty(pk.GetName(), "key name must not be empty")
		s.Assert().NotEmpty(pk.GetPublicKeyPemEnc(), "key PEM must not be empty")
	}
	t.Logf("Red Hat integration found with %d key(s)",
		len(matched.GetCosign().GetPublicKeys()))
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

	bundle := signatures.KeyBundle{
		SchemaVersion: "1.0",
		Keys: []signatures.KeyBundleEntry{
			{Name: "test-key-1", Type: "cosign", PEM: testPublicKeyPEM1},
			{Name: "test-key-2", Type: "cosign", PEM: testPublicKeyPEM2},
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

func (s *RedHatSigningKeySuite) TestWatcherPicksUpV1Bundle() {
	t := s.T()
	ns := namespaces.StackRox
	testCtx, overallCtx, cancel := testContexts(t, "TestWatcherPicksUpV1Bundle", 10*time.Minute)
	defer cancel()

	defer func() {
		s.logf("Cleanup: removing %s env var", watchIntervalEnv)
		s.mustDeleteDeploymentEnvVar(overallCtx, ns, "central", watchIntervalEnv)
		s.waitUntilK8sDeploymentReady(overallCtx, ns, "central")
	}()

	s.logf("Setting %s=%s on central", watchIntervalEnv, shortWatchInterval)
	s.mustSetDeploymentEnvVal(testCtx, ns, "central", "central", watchIntervalEnv, shortWatchInterval)
	s.waitUntilK8sDeploymentReady(testCtx, ns, "central")

	bundle := signatures.KeyBundle{
		SchemaVersion: "1.0",
		Keys: []signatures.KeyBundleEntry{
			{Name: "v1-key-1", Type: "cosign", PEM: testPublicKeyPEM1},
			{Name: "v1-key-unsupported", Type: "pgp", PEM: testPublicKeyPEM2},
		},
	}
	bundleJSON, err := json.Marshal(bundle)
	s.Require().NoError(err)

	b64 := base64.StdEncoding.EncodeToString(bundleJSON)
	writeCmd := fmt.Sprintf("mkdir -p /tmp/redhat-signing-keys && echo %s | base64 -d > /tmp/redhat-signing-keys/bundle.json", b64)

	s.logf("Writing v1.0 mixed-type key bundle to Central pod")
	execInDeployment(t, s.k8s, "central", ns, "sh", "-c", writeCmd)

	defer func() {
		s.logf("Cleanup: removing test bundle file")
		execInDeployment(t, s.k8s, "central", ns, "sh", "-c", "rm -f /tmp/redhat-signing-keys/bundle.json")
	}()

	s.logf("Waiting for watcher to pick up v1.0 bundle and filter non-cosign keys")
	s.waitForIntegrationKeys(testCtx, []string{"v1-key-1"},
		"watcher did not filter non-cosign keys from v1.0 bundle")

	t.Log("Watcher successfully picked up v1.0 bundle and filtered non-cosign keys")
}

func (s *RedHatSigningKeySuite) TestWatcherAcceptsUnknownSchemaVersion() {
	t := s.T()
	ns := namespaces.StackRox
	testCtx, overallCtx, cancel := testContexts(t, "TestWatcherAcceptsUnknownSchemaVersion", 10*time.Minute)
	defer cancel()

	defer func() {
		s.logf("Cleanup: removing %s env var", watchIntervalEnv)
		s.mustDeleteDeploymentEnvVar(overallCtx, ns, "central", watchIntervalEnv)
		s.waitUntilK8sDeploymentReady(overallCtx, ns, "central")
	}()

	s.logf("Setting %s=%s on central", watchIntervalEnv, shortWatchInterval)
	s.mustSetDeploymentEnvVal(testCtx, ns, "central", "central", watchIntervalEnv, shortWatchInterval)
	s.waitUntilK8sDeploymentReady(testCtx, ns, "central")

	bundle := signatures.KeyBundle{
		SchemaVersion: "3.0",
		Keys: []signatures.KeyBundleEntry{
			{Name: "future-key-1", Type: "cosign", PEM: testPublicKeyPEM1},
		},
	}
	bundleJSON, err := json.Marshal(bundle)
	s.Require().NoError(err)

	b64 := base64.StdEncoding.EncodeToString(bundleJSON)
	writeCmd := fmt.Sprintf("mkdir -p /tmp/redhat-signing-keys && echo %s | base64 -d > /tmp/redhat-signing-keys/bundle.json", b64)

	s.logf("Writing bundle with unknown schema version 3.0 to Central pod")
	execInDeployment(t, s.k8s, "central", ns, "sh", "-c", writeCmd)

	defer func() {
		s.logf("Cleanup: removing test bundle file")
		execInDeployment(t, s.k8s, "central", ns, "sh", "-c", "rm -f /tmp/redhat-signing-keys/bundle.json")
	}()

	s.logf("Waiting for watcher to accept bundle with unknown schema version")
	s.waitForIntegrationKeys(testCtx, []string{"future-key-1"},
		"watcher did not accept bundle with unknown schema version")

	t.Log("Watcher accepted bundle with unknown schema version 3.0 and extracted cosign key")
}

func (s *RedHatSigningKeySuite) TestWatcherBadBundleDoesNotOverwriteExistingKeys() {
	t := s.T()
	ns := namespaces.StackRox
	testCtx, overallCtx, cancel := testContexts(t, "TestWatcherBadBundleDoesNotOverwriteExistingKeys", 10*time.Minute)
	defer cancel()

	defer func() {
		s.logf("Cleanup: removing %s env var", watchIntervalEnv)
		s.mustDeleteDeploymentEnvVar(overallCtx, ns, "central", watchIntervalEnv)
		s.waitUntilK8sDeploymentReady(overallCtx, ns, "central")
	}()

	s.logf("Setting %s=%s on central", watchIntervalEnv, shortWatchInterval)
	s.mustSetDeploymentEnvVal(testCtx, ns, "central", "central", watchIntervalEnv, shortWatchInterval)
	s.waitUntilK8sDeploymentReady(testCtx, ns, "central")

	goodBundle := signatures.KeyBundle{
		SchemaVersion: "1.0",
		Keys: []signatures.KeyBundleEntry{
			{Name: "good-key", Type: "cosign", PEM: testPublicKeyPEM1},
		},
	}
	goodJSON, err := json.Marshal(goodBundle)
	s.Require().NoError(err)

	b64 := base64.StdEncoding.EncodeToString(goodJSON)
	writeCmd := fmt.Sprintf("mkdir -p /tmp/redhat-signing-keys && echo %s | base64 -d > /tmp/redhat-signing-keys/bundle.json", b64)
	s.logf("Writing good bundle to establish baseline keys")
	execInDeployment(t, s.k8s, "central", ns, "sh", "-c", writeCmd)

	s.waitForIntegrationKeys(testCtx, []string{"good-key"},
		"watcher did not pick up good bundle")
	t.Log("Baseline established: integration has good-key")

	badBundle := signatures.KeyBundle{
		SchemaVersion: "1.0",
		Keys: []signatures.KeyBundleEntry{
			{Name: "pgp-only", Type: "pgp", PEM: testPublicKeyPEM2},
		},
	}
	badJSON, err := json.Marshal(badBundle)
	s.Require().NoError(err)

	b64Bad := base64.StdEncoding.EncodeToString(badJSON)
	writeBadCmd := fmt.Sprintf("echo %s | base64 -d > /tmp/redhat-signing-keys/bundle.json", b64Bad)
	s.logf("Overwriting with bad bundle (only unsupported key types)")
	execInDeployment(t, s.k8s, "central", ns, "sh", "-c", writeBadCmd)

	s.logf("Waiting two watcher cycles to confirm keys are NOT overwritten")
	time.Sleep(25 * time.Second)

	s.waitForIntegrationKeys(testCtx, []string{"good-key"},
		"bad bundle overwrote existing keys — integration should still have good-key")

	defer func() {
		s.logf("Cleanup: removing test bundle file")
		execInDeployment(t, s.k8s, "central", ns, "sh", "-c", "rm -f /tmp/redhat-signing-keys/bundle.json")
	}()

	t.Log("Bad bundle correctly did not overwrite existing keys")
}

func (s *RedHatSigningKeySuite) TestUpdaterDownloadsBundleFromHTTP() {
	t := s.T()
	ns := namespaces.StackRox
	testCtx, overallCtx, cancel := testContexts(t, "TestUpdaterDownloadsBundleFromHTTP", 10*time.Minute)
	defer cancel()

	configMapName := "rh-signing-key-bundle-test"
	deploymentName := "key-bundle-server"
	bundleURLEnv := "ROX_REDHAT_SIGNING_KEY_BUNDLE_URL"
	updateIntervalEnv := "ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL"

	bundle := signatures.KeyBundle{
		SchemaVersion: "1.0",
		Keys: []signatures.KeyBundleEntry{
			{Name: "updater-key-1", Type: "cosign", PEM: testPublicKeyPEM1},
			{Name: "updater-key-2", Type: "cosign", PEM: testPublicKeyPEM2},
		},
	}
	bundleJSON, err := json.Marshal(bundle)
	s.Require().NoError(err)

	s.logf("Creating ConfigMap %q with key bundle", configMapName)
	s.ensureConfigMapExists(testCtx, ns, configMapName, map[string]string{
		"bundle.json": string(bundleJSON),
	})

	defer func() {
		s.logf("Cleanup: deleting ConfigMap %q", configMapName)
		_ = s.k8s.CoreV1().ConfigMaps(ns).Delete(overallCtx, configMapName, metaV1.DeleteOptions{})
	}()

	s.logf("Creating nginx deployment %q", deploymentName)
	nginxDeploy := &appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      deploymentName,
			Namespace: ns,
			Labels:    map[string]string{"app": deploymentName},
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: pointers.Int32(1),
			Selector: &metaV1.LabelSelector{
				MatchLabels: map[string]string{"app": deploymentName},
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{"app": deploymentName},
				},
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{{
						Name:  "nginx",
						Image: "docker.io/nginxinc/nginx-unprivileged:alpine",
						Ports: []coreV1.ContainerPort{{ContainerPort: 8080, Protocol: coreV1.ProtocolTCP}},
						VolumeMounts: []coreV1.VolumeMount{{
							Name:      "bundle",
							MountPath: "/usr/share/nginx/html",
							ReadOnly:  true,
						}},
					}},
					Volumes: []coreV1.Volume{{
						Name: "bundle",
						VolumeSource: coreV1.VolumeSource{
							ConfigMap: &coreV1.ConfigMapVolumeSource{
								LocalObjectReference: coreV1.LocalObjectReference{Name: configMapName},
							},
						},
					}},
				},
			},
		},
	}
	_, err = s.k8s.AppsV1().Deployments(ns).Create(testCtx, nginxDeploy, metaV1.CreateOptions{})
	s.Require().NoError(err, "creating nginx deployment")

	defer func() {
		s.logf("Cleanup: deleting deployment %q", deploymentName)
		_ = s.k8s.AppsV1().Deployments(ns).Delete(overallCtx, deploymentName, metaV1.DeleteOptions{})
	}()

	s.waitUntilK8sDeploymentReady(testCtx, ns, deploymentName)

	s.createService(testCtx, ns, deploymentName,
		map[string]string{"app": deploymentName},
		map[int32]int32{80: 8080})

	defer func() {
		s.logf("Cleanup: deleting service %q", deploymentName)
		_ = s.k8s.CoreV1().Services(ns).Delete(overallCtx, deploymentName, metaV1.DeleteOptions{})
	}()

	bundleURL := fmt.Sprintf("http://%s.%s.svc/bundle.json", deploymentName, ns)
	s.logf("Bundle URL: %s", bundleURL)

	// Service endpoints may not be routable immediately after the deployment
	// is marked ready. Verify reachability before restarting Central, because
	// the updater fires its first download immediately on startup and the
	// retry interval is clamped to 5 minutes.
	s.logf("Verifying bundle URL is reachable from within the cluster")
	mustEventually(t, testCtx, func() error {
		podList, err := s.k8s.CoreV1().Pods(ns).List(testCtx, metaV1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", deploymentName),
		})
		if err != nil {
			return fmt.Errorf("listing pods: %w", err)
		}
		if len(podList.Items) == 0 {
			return fmt.Errorf("no pods found for deployment %q", deploymentName)
		}
		out, err := exec.CommandContext(testCtx, "kubectl", "exec", "-n", ns, podList.Items[0].Name, "--", "wget", "-q", "-O", "/dev/null", bundleURL).CombinedOutput()
		if err != nil {
			return fmt.Errorf("wget failed: %s: %w", string(out), err)
		}
		return nil
	}, 2*time.Second, "bundle URL not yet reachable")
	s.logf("Bundle URL is reachable")

	// The watcher must poll frequently so it picks up the file the updater writes.
	defer func() {
		s.logf("Cleanup: removing updater env vars from Central")
		s.mustDeleteDeploymentEnvVar(overallCtx, ns, "central", bundleURLEnv)
		s.mustDeleteDeploymentEnvVar(overallCtx, ns, "central", updateIntervalEnv)
		s.mustDeleteDeploymentEnvVar(overallCtx, ns, "central", watchIntervalEnv)
		s.waitUntilK8sDeploymentReady(overallCtx, ns, "central")
	}()

	s.logf("Setting %s, %s, and %s on central", bundleURLEnv, updateIntervalEnv, watchIntervalEnv)
	s.mustSetDeploymentEnvVal(testCtx, ns, "central", "central", bundleURLEnv, bundleURL)
	// The interval gets clamped to 5m minimum, but the first download runs immediately on startup.
	s.mustSetDeploymentEnvVal(testCtx, ns, "central", "central", updateIntervalEnv, "10s")
	s.mustSetDeploymentEnvVal(testCtx, ns, "central", "central", watchIntervalEnv, shortWatchInterval)
	s.waitUntilK8sDeploymentReady(testCtx, ns, "central")

	s.logf("Waiting for updater to download the bundle and watcher to upsert keys")
	s.waitForIntegrationKeys(testCtx, []string{"updater-key-1", "updater-key-2"},
		"updater did not download and apply the bundle")

	t.Log("Updater successfully downloaded bundle and applied 2 keys")
}
