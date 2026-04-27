package manager

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	admission "k8s.io/api/admission/v1"
	apps "k8s.io/api/apps/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sTypes "k8s.io/apimachinery/pkg/types"
)

// --- Fake Image Service Client ---

type coalescingBenchImageClient struct {
	latency      time.Duration
	callCount    atomic.Int64
	mu           sync.Mutex
	callsByImage map[string]int64
}

func newCoalescingBenchImageClient(latency time.Duration) *coalescingBenchImageClient {
	return &coalescingBenchImageClient{
		latency:      latency,
		callsByImage: make(map[string]int64),
	}
}

func (c *coalescingBenchImageClient) GetImage(ctx context.Context, req *sensor.GetImageRequest, _ ...grpc.CallOption) (*sensor.GetImageResponse, error) {
	c.callCount.Add(1)

	imageID := req.GetImage().GetId()
	if imageID == "" {
		hash := sha256.Sum256([]byte(req.GetImage().GetName().GetFullName()))
		imageID = fmt.Sprintf("sha256:%x", hash)
	}

	c.mu.Lock()
	c.callsByImage[imageID]++
	c.mu.Unlock()

	if c.latency > 0 {
		select {
		case <-time.After(c.latency):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &sensor.GetImageResponse{
		Image: &storage.Image{
			Id: imageID,
			Name: &storage.ImageName{
				Registry: req.GetImage().GetName().GetRegistry(),
				Remote:   req.GetImage().GetName().GetRemote(),
				Tag:      req.GetImage().GetName().GetTag(),
				FullName: req.GetImage().GetName().GetFullName(),
			},
			Scan: &storage.ImageScan{
				ScanTime: nil,
			},
		},
	}, nil
}

func (c *coalescingBenchImageClient) CallCount() int64 {
	return c.callCount.Load()
}

func (c *coalescingBenchImageClient) CallsByImage() map[string]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make(map[string]int64, len(c.callsByImage))
	for k, v := range c.callsByImage {
		result[k] = v
	}
	return result
}

func (c *coalescingBenchImageClient) Reset() {
	c.callCount.Store(0)
	c.mu.Lock()
	c.callsByImage = make(map[string]int64)
	c.mu.Unlock()
}

// --- No-op Deployment Service Client ---

type noopDeploymentServiceClient struct{}

func (noopDeploymentServiceClient) GetDeploymentForPod(context.Context, *sensor.GetDeploymentForPodRequest, ...grpc.CallOption) (*storage.Deployment, error) {
	return nil, nil
}

// --- Data Generators ---

type imageRef struct {
	id       string
	registry string
	remote   string
	tag      string
	fullName string
}

func generateImagePool(n int) []imageRef {
	pool := make([]imageRef, n)
	for i := range n {
		tag := fmt.Sprintf("v1.%d.0", i)
		remote := fmt.Sprintf("library/image-%d", i)
		fullName := fmt.Sprintf("docker.io/%s:%s", remote, tag)
		hash := sha256.Sum256([]byte(fullName))
		pool[i] = imageRef{
			id:       fmt.Sprintf("sha256:%x", hash),
			registry: "docker.io",
			remote:   remote,
			tag:      tag,
			fullName: fullName,
		}
	}
	return pool
}

func generateDeployment(name, namespace string, images []imageRef) *apps.Deployment {
	containers := make([]core.Container, len(images))
	for i, img := range images {
		containers[i] = core.Container{
			Name:  fmt.Sprintf("container-%d", i),
			Image: img.fullName,
		}
	}
	replicas := int32(1)
	return &apps.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       k8sTypes.UID(uuid.NewV4().String()),
		},
		Spec: apps.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: core.PodSpec{
					Containers: containers,
				},
			},
		},
	}
}

func deploymentToAdmissionRequest(dep *apps.Deployment) *admission.AdmissionRequest {
	raw, err := json.Marshal(dep)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal deployment: %v", err))
	}
	return &admission.AdmissionRequest{
		UID: k8sTypes.UID(uuid.NewV4().String()),
		Kind: metav1.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		},
		Resource: metav1.GroupVersionResource{
			Group:    "apps",
			Version:  "v1",
			Resource: "deployments",
		},
		Name:      dep.Name,
		Namespace: dep.Namespace,
		Operation: admission.Create,
		Object:    runtime.RawExtension{Raw: raw},
		UserInfo: authenticationv1.UserInfo{
			Username: "test-user",
			Groups:   []string{"system:authenticated"},
		},
	}
}

// workload holds pre-generated admission requests and associated metadata.
type workload struct {
	requests     []*admission.AdmissionRequest
	uniqueImages int
	imgsPerDep   int
}

func generateWorkload(rng *rand.Rand, nDeployments, imgsPerDep, uniqueImages int) workload {
	pool := generateImagePool(uniqueImages)
	requests := make([]*admission.AdmissionRequest, nDeployments)
	for i := range nDeployments {
		images := make([]imageRef, imgsPerDep)
		for j := range imgsPerDep {
			images[j] = pool[rng.IntN(len(pool))]
		}
		dep := generateDeployment(
			fmt.Sprintf("deploy-%d", i),
			"bench-ns",
			images,
		)
		requests[i] = deploymentToAdmissionRequest(dep)
	}
	return workload{
		requests:     requests,
		uniqueImages: uniqueImages,
		imgsPerDep:   imgsPerDep,
	}
}

// --- Benchmark Helpers ---

func cachingBenchSettings() *sensor.AdmissionControlSettings {
	return &sensor.AdmissionControlSettings{
		ClusterId: uuid.NewDummy().String(),
		ClusterConfig: &storage.DynamicClusterConfig{
			AdmissionControllerConfig: &storage.AdmissionControllerConfig{
				Enabled:          true,
				EnforceOnUpdates: false,
				TimeoutSeconds:   10,
				ScanInline:       true,
			},
		},
		EnforcedDeployTimePolicies: &storage.PolicyList{Policies: []*storage.Policy{
			imageAgePolicy(),
		}},
		RuntimePolicies: &storage.PolicyList{},
	}
}

func imageAgePolicy() *storage.Policy {
	return &storage.Policy{
		Id:              uuid.NewV4().String(),
		PolicyVersion:   "1.1",
		Name:            "Image Age",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Severity:        storage.Severity_LOW_SEVERITY,
		Categories:      []string{"DevOps Best Practices"},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Image Age",
						Values: []*storage.PolicyValue{
							{Value: "30"},
						},
					},
				},
			},
		},
		EnforcementActions: []storage.EnforcementAction{
			storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
		},
	}
}

func setupManager(b *testing.B, imgClient sensor.ImageServiceClient) *manager {
	b.Helper()
	logging.SetGlobalLogLevel(zapcore.FatalLevel)
	mgr := NewManager("stackrox", 200*size.MB, true, imgClient, noopDeploymentServiceClient{})
	mgr.ProcessNewSettings(cachingBenchSettings())
	mgr.Start()
	b.Cleanup(func() {
		mgr.Stop()
		logging.SetGlobalLogLevel(zapcore.InfoLevel)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(b, mgr.Sync(ctx))
	return mgr
}

// drainAlerts reads any pending alerts so the alertsC channel doesn't block.
func drainAlerts(mgr *manager, done <-chan struct{}) {
	go func() {
		for {
			select {
			case <-done:
				return
			case <-mgr.alertsC:
			}
		}
	}()
}

// --- Latency Statistics ---

func percentile(durations []time.Duration, p float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

// --- Benchmarks ---

func BenchmarkAdmissionReview_BaselineColdCache(b *testing.B) {
	imgClient := newCoalescingBenchImageClient(50 * time.Millisecond)
	mgr := setupManager(b, imgClient)
	done := make(chan struct{})
	defer close(done)
	drainAlerts(mgr, done)

	rng := rand.New(rand.NewPCG(42, 0))
	wl := generateWorkload(rng, 100, 3, 300)

	for b.Loop() {
		imgClient.Reset()
		mgr.imageCache.Purge()
		mgr.imageNameToImageCacheKey.Purge()

		for _, req := range wl.requests {
			_, _ = mgr.HandleValidate(req)
		}

		b.ReportMetric(float64(imgClient.CallCount()), "rpc_count")
		b.ReportMetric(float64(imgClient.CallCount())/float64(len(wl.requests)), "rpcs/review")
	}
}

func BenchmarkAdmissionReview_ConcurrentBurst_LowDiversity(b *testing.B) {
	imgClient := newCoalescingBenchImageClient(100 * time.Millisecond)
	mgr := setupManager(b, imgClient)
	done := make(chan struct{})
	defer close(done)
	drainAlerts(mgr, done)

	rng := rand.New(rand.NewPCG(42, 0))
	wl := generateWorkload(rng, 1000, 3, 50)

	for b.Loop() {
		imgClient.Reset()
		mgr.imageCache.Purge()
		mgr.imageNameToImageCacheKey.Purge()

		latencies := runConcurrentReviews(mgr, wl.requests)

		b.ReportMetric(float64(imgClient.CallCount()), "rpc_count")
		b.ReportMetric(float64(imgClient.CallCount())/float64(len(wl.requests)), "rpcs/review")
		b.ReportMetric(float64(percentile(latencies, 0.50).Milliseconds()), "p50_ms")
		b.ReportMetric(float64(percentile(latencies, 0.99).Milliseconds()), "p99_ms")
	}
}

func BenchmarkAdmissionReview_ConcurrentBurst_MediumDiversity(b *testing.B) {
	imgClient := newCoalescingBenchImageClient(100 * time.Millisecond)
	mgr := setupManager(b, imgClient)
	done := make(chan struct{})
	defer close(done)
	drainAlerts(mgr, done)

	rng := rand.New(rand.NewPCG(42, 0))
	wl := generateWorkload(rng, 1000, 3, 200)

	for b.Loop() {
		imgClient.Reset()
		mgr.imageCache.Purge()
		mgr.imageNameToImageCacheKey.Purge()

		latencies := runConcurrentReviews(mgr, wl.requests)

		b.ReportMetric(float64(imgClient.CallCount()), "rpc_count")
		b.ReportMetric(float64(imgClient.CallCount())/float64(len(wl.requests)), "rpcs/review")
		b.ReportMetric(float64(percentile(latencies, 0.50).Milliseconds()), "p50_ms")
		b.ReportMetric(float64(percentile(latencies, 0.99).Milliseconds()), "p99_ms")
	}
}

func BenchmarkAdmissionReview_ConcurrentBurst_HighDiversity(b *testing.B) {
	imgClient := newCoalescingBenchImageClient(100 * time.Millisecond)
	mgr := setupManager(b, imgClient)
	done := make(chan struct{})
	defer close(done)
	drainAlerts(mgr, done)

	rng := rand.New(rand.NewPCG(42, 0))
	wl := generateWorkload(rng, 1000, 3, 2000)

	for b.Loop() {
		imgClient.Reset()
		mgr.imageCache.Purge()
		mgr.imageNameToImageCacheKey.Purge()

		latencies := runConcurrentReviews(mgr, wl.requests)

		b.ReportMetric(float64(imgClient.CallCount()), "rpc_count")
		b.ReportMetric(float64(imgClient.CallCount())/float64(len(wl.requests)), "rpcs/review")
		b.ReportMetric(float64(percentile(latencies, 0.50).Milliseconds()), "p50_ms")
		b.ReportMetric(float64(percentile(latencies, 0.99).Milliseconds()), "p99_ms")
	}
}

func BenchmarkAdmissionReview_WarmCache(b *testing.B) {
	imgClient := newCoalescingBenchImageClient(100 * time.Millisecond)
	mgr := setupManager(b, imgClient)
	done := make(chan struct{})
	defer close(done)
	drainAlerts(mgr, done)

	rng := rand.New(rand.NewPCG(42, 0))
	wl := generateWorkload(rng, 1000, 3, 100)

	// Warm the cache by running all reviews once sequentially.
	for _, req := range wl.requests {
		_, _ = mgr.HandleValidate(req)
	}
	imgClient.Reset()

	for b.Loop() {
		// Cache stays warm across iterations.
		imgClient.Reset()

		latencies := runConcurrentReviews(mgr, wl.requests)

		b.ReportMetric(float64(imgClient.CallCount()), "rpc_count")
		b.ReportMetric(float64(percentile(latencies, 0.50).Milliseconds()), "p50_ms")
		b.ReportMetric(float64(percentile(latencies, 0.99).Milliseconds()), "p99_ms")
	}
}

func BenchmarkAdmissionReview_ScaleSweep(b *testing.B) {
	deployCounts := []int{100, 1000, 5000}
	uniqueRatios := []struct {
		name  string
		ratio float64
	}{
		{"1pct_unique", 0.01},
		{"10pct_unique", 0.10},
		{"50pct_unique", 0.50},
		{"100pct_unique", 1.0},
	}

	for _, nDeps := range deployCounts {
		for _, ur := range uniqueRatios {
			imgsPerDep := 3
			totalRefs := nDeps * imgsPerDep
			uniqueImgs := max(1, int(float64(totalRefs)*ur.ratio))

			name := fmt.Sprintf("deps=%d/%s/imgs_per_dep=%d/unique=%d",
				nDeps, ur.name, imgsPerDep, uniqueImgs)

			b.Run(name, func(b *testing.B) {
				imgClient := newCoalescingBenchImageClient(100 * time.Millisecond)
				mgr := setupManager(b, imgClient)
				done := make(chan struct{})
				defer close(done)
				drainAlerts(mgr, done)

				rng := rand.New(rand.NewPCG(42, 0))
				wl := generateWorkload(rng, nDeps, imgsPerDep, uniqueImgs)

				for b.Loop() {
					imgClient.Reset()
					mgr.imageCache.Purge()
					mgr.imageNameToImageCacheKey.Purge()

					latencies := runConcurrentReviews(mgr, wl.requests)

					b.ReportMetric(float64(imgClient.CallCount()), "rpc_count")
					b.ReportMetric(float64(imgClient.CallCount())/float64(len(wl.requests)), "rpcs/review")
					b.ReportMetric(float64(percentile(latencies, 0.50).Milliseconds()), "p50_ms")
					b.ReportMetric(float64(percentile(latencies, 0.99).Milliseconds()), "p99_ms")
				}
			})
		}
	}
}

// --- Concurrent Review Runner ---

func runConcurrentReviews(mgr *manager, requests []*admission.AdmissionRequest) []time.Duration {
	latencies := make([]time.Duration, len(requests))
	var wg sync.WaitGroup
	wg.Add(len(requests))
	for i, req := range requests {
		go func(idx int, r *admission.AdmissionRequest) {
			defer wg.Done()
			start := time.Now()
			_, _ = mgr.HandleValidate(r)
			latencies[idx] = time.Since(start)
		}(i, req)
	}
	wg.Wait()
	return latencies
}
