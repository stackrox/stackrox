package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/uuid"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	admission "k8s.io/api/admission/v1"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

// --- Fake gRPC clients ---

type benchImageServiceClient struct {
	latency   time.Duration
	callCount atomic.Int64
}

func (c *benchImageServiceClient) GetImage(ctx context.Context, req *sensor.GetImageRequest, _ ...grpc.CallOption) (*sensor.GetImageResponse, error) {
	c.callCount.Add(1)
	select {
	case <-time.After(c.latency):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &sensor.GetImageResponse{
		Image: violatingImage(req.GetImage()),
	}, nil
}

func (c *benchImageServiceClient) Reset()           { c.callCount.Store(0) }
func (c *benchImageServiceClient) CallCount() int64 { return c.callCount.Load() }

type benchDeploymentServiceClient struct{}

func (benchDeploymentServiceClient) GetDeploymentForPod(_ context.Context, _ *sensor.GetDeploymentForPodRequest, _ ...grpc.CallOption) (*storage.Deployment, error) {
	return nil, nil
}

// --- Violating image builder ---

func violatingImage(ci *storage.ContainerImage) *storage.Image {
	oneYearAgo := time.Now().Add(-365 * 24 * time.Hour)
	return &storage.Image{
		Id:   ci.GetId(),
		Name: ci.GetName(),
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: protocompat.ConvertTimeToTimestampOrNil(&oneYearAgo),
				Layers: []*storage.ImageLayer{
					{
						Instruction: "COPY",
						Value:       "app /app",
					},
				},
			},
		},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "openssl",
					Version: "1.1.1k",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:        "CVE-2021-99999",
							Cvss:       9.8,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.1.1l"},
							Severity:   storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
						},
					},
				},
			},
			ScanTime: protocompat.ConvertTimeToTimestampOrNil(&oneYearAgo),
		},
		SetCves: &storage.Image_Cves{Cves: 1},
	}
}

// --- AdmissionRequest generators ---

var benchImages = []string{
	"docker.io/library/nginx@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	"docker.io/library/redis@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	"gcr.io/myproject/app@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
}

func makeAdmissionRequest(name, namespace string, images []string, privileged bool) *admission.AdmissionRequest {
	containers := make([]coreV1.Container, len(images))
	for i, img := range images {
		containers[i] = coreV1.Container{
			Name:  fmt.Sprintf("container-%d", i),
			Image: img,
		}
		if privileged {
			containers[i].SecurityContext = &coreV1.SecurityContext{
				Privileged: pointer.Bool(true),
			}
		}
	}

	dep := &appsV1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":  "bench-app",
				"env":  "production",
				"team": "platform",
			},
		},
		Spec: appsV1.DeploymentSpec{
			Template: coreV1.PodTemplateSpec{
				Spec: coreV1.PodSpec{
					Containers: containers,
				},
			},
		},
	}

	raw, err := json.Marshal(dep)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal deployment: %v", err))
	}

	return &admission.AdmissionRequest{
		UID:       types.UID(uuid.NewV4().String()),
		Operation: admission.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		},
		Namespace: namespace,
		Name:      name,
		Object:    runtime.RawExtension{Raw: raw},
		DryRun:    pointer.Bool(true),
	}
}

func makeAdmissionRequests(deploymentCount int, violationPct float64, images []string) []*admission.AdmissionRequest {
	violatingCount := int(float64(deploymentCount) * violationPct)
	reqs := make([]*admission.AdmissionRequest, deploymentCount)
	for i := 0; i < deploymentCount; i++ {
		privileged := i < violatingCount
		reqs[i] = makeAdmissionRequest(fmt.Sprintf("deploy-%d", i), "bench", images, privileged)
	}
	return reqs
}

// --- Policy generators ---

var fastPathFields = []string{
	"Privileged Container",
	"Image Tag",
	"Image Remote",
	"Image Registry",
	"Volume Name",
	"Volume Type",
	"Volume Destination",
	"Host Network",
	"Host PID",
	"Container Name",
}

var slowPathFields = []string{
	"CVE",
	"CVSS",
	"Image Age",
	"Image Scan Age",
	"Image Component",
	"Fixed By",
	"Severity",
	"Dockerfile Line",
	"Unscanned Image",
	"Image Signature Verified By",
}

// fastPathPolicyValue returns values that do NOT match the benchmark
// deployments. Only PrivilegedContainer is designed to fire -- and only
// when the deployment has privileged=true. All other fast-path policies
// use values that won't match, so they exist for compilation/evaluation
// overhead but don't produce alerts.
func fastPathPolicyValue(fieldName string) string {
	switch fieldName {
	case "Privileged Container":
		return "true"
	case "Image Tag":
		return "nonexistent-tag-xyz"
	case "Image Remote":
		return "no-such-remote/.*"
	case "Image Registry":
		return "fake-registry.example.com"
	case "Volume Name":
		return "no-such-volume"
	case "Volume Type":
		return "Host Path"
	case "Volume Destination":
		return "/nonexistent/path/xyz"
	case "Host Network":
		return "true"
	case "Host PID":
		return "true"
	case "Container Name":
		return "no-such-container-.*"
	default:
		return "no-match-value"
	}
}

// slowPathPolicyValue returns values that WILL match the violating image
// returned by benchImageServiceClient, so slow-path policies actually
// produce alerts when enrichment data is available.
func slowPathPolicyValue(fieldName string) string {
	switch fieldName {
	case "CVE":
		return "CVE-2021-.*"
	case "CVSS":
		return ">= 7.0"
	case "Image Age":
		return "30"
	case "Image Scan Age":
		return "7"
	case "Image Component":
		return "openssl="
	case "Fixed By":
		return ".*"
	case "Severity":
		return ">= CRITICAL"
	case "Dockerfile Line":
		return "COPY=.*"
	case "Unscanned Image":
		return "true"
	case "Image Signature Verified By":
		return "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000000"
	default:
		return ".*"
	}
}

func generatePolicies(prefix string, n int, fields []string, valueFn func(string) string) []*storage.Policy {
	policies := make([]*storage.Policy, n)
	for i := 0; i < n; i++ {
		field := fields[i%len(fields)]
		policies[i] = &storage.Policy{
			Id:              uuid.NewV4().String(),
			Name:            fmt.Sprintf("%s-policy-%d", prefix, i),
			Disabled:        false,
			PolicyVersion:   "1.1",
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
			EnforcementActions: []storage.EnforcementAction{
				storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
			},
			PolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       field,
							BooleanOperator: storage.BooleanOperator_OR,
							Values: []*storage.PolicyValue{
								{Value: valueFn(field)},
							},
						},
					},
				},
			},
		}
	}
	return policies
}

func fastPathPolicies(n int) []*storage.Policy {
	return generatePolicies("fast", n, fastPathFields, fastPathPolicyValue)
}

func slowPathPolicies(n int) []*storage.Policy {
	return generatePolicies("slow", n, slowPathFields, slowPathPolicyValue)
}

// --- Settings generator ---

func benchSettings(fastPolicies, slowPolicies []*storage.Policy) *sensor.AdmissionControlSettings {
	allPolicies := make([]*storage.Policy, 0, len(fastPolicies)+len(slowPolicies))
	allPolicies = append(allPolicies, fastPolicies...)
	allPolicies = append(allPolicies, slowPolicies...)

	return &sensor.AdmissionControlSettings{
		ClusterId: "bench-cluster-id",
		EnforcedDeployTimePolicies: &storage.PolicyList{
			Policies: allPolicies,
		},
		ClusterConfig: &storage.DynamicClusterConfig{
			AdmissionControllerConfig: &storage.AdmissionControllerConfig{
				Enabled:        true,
				ScanInline:     true,
				TimeoutSeconds: 10,
			},
		},
		Timestamp: timestamppb.Now(),
	}
}

// --- Pruning helpers ---

func maxDeployments(fast, slow int) int {
	if slow > fast {
		return 1000
	}
	return 10000
}

func shouldSweepViolationPct(fast int) bool {
	return fast > 0
}

func validatePoliciesCompile(b *testing.B, policies []*storage.Policy, label string) {
	b.Helper()
	for _, p := range policies {
		if _, err := detection.CompilePolicy(p, nil, nil); err != nil {
			b.Fatalf("%s policy %q failed to compile: %v", label, p.GetName(), err)
		}
	}
}

// --- Benchmark ---

func BenchmarkDetectionSplit(b *testing.B) {
	logging.SetGlobalLogLevel(zapcore.FatalLevel)
	b.Cleanup(func() { logging.SetGlobalLogLevel(zapcore.InfoLevel) })

	policyMixes := []struct{ fast, slow int }{
		{40, 0},
		{30, 10},
		{20, 20},
		{10, 30},
		{0, 40},
	}
	deploymentCounts := []int{1, 10, 100, 1000, 10000}
	violationPcts := []float64{0.0, 0.25, 0.50, 0.75, 1.0}

	for _, mix := range policyMixes {
		for _, depCount := range deploymentCounts {
			if depCount > maxDeployments(mix.fast, mix.slow) {
				continue
			}

			vPcts := violationPcts
			if !shouldSweepViolationPct(mix.fast) {
				vPcts = []float64{0.0}
			}

			for _, vPct := range vPcts {
				name := fmt.Sprintf("fast=%d/slow=%d/deps=%d/viol=%.0f%%", mix.fast, mix.slow, depCount, vPct*100)
				b.Run(name, func(b *testing.B) {
					imgClient := &benchImageServiceClient{latency: 100 * time.Millisecond}
					mgr := NewManager("bench-ns", 20*size.MB, true, imgClient, &benchDeploymentServiceClient{})

					fp := fastPathPolicies(mix.fast)
					sp := slowPathPolicies(mix.slow)
					validatePoliciesCompile(b, fp, "fast-path")
					validatePoliciesCompile(b, sp, "slow-path")
					mgr.ProcessNewSettings(benchSettings(fp, sp))

					reqs := makeAdmissionRequests(depCount, vPct, benchImages)

					for b.Loop() {
						imgClient.Reset()
						mgr.imageCache.Purge()

						totalDenials := 0
						for _, req := range reqs {
							resp, err := mgr.HandleValidate(req)
							if err != nil {
								b.Fatalf("HandleValidate error: %v", err)
							}
							if resp != nil && !resp.Allowed {
								totalDenials++
							}
						}

						rpcs := imgClient.CallCount()
						b.ReportMetric(float64(rpcs), "rpc_count")
						if depCount > 0 {
							b.ReportMetric(float64(rpcs)/float64(depCount), "rpc_per_review")
						}
						b.ReportMetric(float64(totalDenials), "total_denials")
					}
				})
			}
		}
	}
}
