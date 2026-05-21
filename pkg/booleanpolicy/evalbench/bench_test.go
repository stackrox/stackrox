package evalbench

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/require"
)

const benchIterations = 10000

func benchImage(maxBaseLayerIdx int32, numComponents, numLayers int) *storage.Image {
	components := make([]*storage.EmbeddedImageScanComponent, numComponents)
	for i := range components {
		comp := &storage.EmbeddedImageScanComponent{
			Name:    fmt.Sprintf("comp-%d", i),
			Version: fmt.Sprintf("%d.0.0", i),
			Vulns: []*storage.EmbeddedVulnerability{{
				Cve:      fmt.Sprintf("CVE-2024-%04d", i),
				Cvss:     5.0,
				Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
			}},
		}
		if maxBaseLayerIdx >= 0 {
			layerIdx := int32(i) * int32(numLayers) / int32(numComponents)
			comp.HasLayerIndex = &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: layerIdx}
		}
		components[i] = comp
	}

	layers := make([]*storage.ImageLayer, numLayers)
	for i := range layers {
		instr := "RUN"
		if i == 0 {
			instr = "ADD"
		}
		layers[i] = &storage.ImageLayer{
			Instruction: instr,
			Value:       fmt.Sprintf("layer-%d command", i),
			Created:     protocompat.TimestampNow(),
		}
	}

	img := &storage.Image{
		Id: "sha256:BENCH",
		Name: &storage.ImageName{
			Registry: "docker.io",
			Remote:   "library/bench",
			Tag:      "latest",
			FullName: "docker.io/library/bench:latest",
		},
		Scan: &storage.ImageScan{
			ScanTime:   protocompat.TimestampNow(),
			Components: components,
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{Layers: layers},
		},
	}

	if maxBaseLayerIdx >= 0 {
		img.BaseImageInfo = []*storage.BaseImageInfo{{MaxLayerIndex: maxBaseLayerIdx}}
	}
	return img
}

func makeED(img *storage.Image) booleanpolicy.EnhancedDeployment {
	dep := &storage.Deployment{
		Id: "d1", Name: "d", Namespace: "ns", ClusterId: "c1", ClusterName: "c",
		Containers: []*storage.Container{{
			Name:            "c0",
			Image:           types.ToContainerImage(img),
			SecurityContext: &storage.SecurityContext{Privileged: true},
		}},
	}
	return booleanpolicy.EnhancedDeployment{
		Deployment: dep,
		Images:     []*storage.Image{img},
		NetworkPoliciesApplied: &augmentedobjs.NetworkPoliciesApplied{
			HasIngressNetworkPolicy: true,
			HasEgressNetworkPolicy:  true,
		},
	}
}

func makePolicy(filter *storage.EvaluationFilter, groups ...*storage.PolicyGroup) *storage.Policy {
	return &storage.Policy{
		Id: "bench", Name: "bench",
		PolicyVersion:    policyversion.CurrentVersion().String(),
		EventSource:      storage.EventSource_NOT_APPLICABLE,
		LifecycleStages:  []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		EvaluationFilter: filter,
		PolicySections:   []*storage.PolicySection{{PolicyGroups: groups}},
	}
}

func cvssGroup() *storage.PolicyGroup {
	return &storage.PolicyGroup{
		FieldName: fieldnames.CVSS,
		Values:    []*storage.PolicyValue{{Value: ">= 3"}},
	}
}

func measure(matcher booleanpolicy.DeploymentMatcher, ed booleanpolicy.EnhancedDeployment, cold bool) (nsPerOp int64, allocsPerOp, heapPerOp uint64, violations int) {
	n := benchIterations

	if cold {
		v, _ := matcher.MatchDeployment(nil, ed)
		for i := 0; i < 100; i++ {
			matcher.MatchDeployment(nil, ed)
		}
		debug.SetGCPercent(-1)
		runtime.GC()
		var before runtime.MemStats
		runtime.ReadMemStats(&before)
		start := time.Now()
		for i := 0; i < n; i++ {
			matcher.MatchDeployment(nil, ed)
		}
		elapsed := time.Since(start)
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		debug.SetGCPercent(100)
		runtime.GC()
		return elapsed.Nanoseconds() / int64(n),
			(after.Mallocs - before.Mallocs) / uint64(n),
			(after.HeapAlloc - before.HeapAlloc) / uint64(n),
			len(v.AlertViolations)
	}

	var cache booleanpolicy.CacheReceptacle
	v, _ := matcher.MatchDeployment(&cache, ed)
	for i := 0; i < 100; i++ {
		matcher.MatchDeployment(&cache, ed)
	}
	debug.SetGCPercent(-1)
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	for i := 0; i < n; i++ {
		matcher.MatchDeployment(&cache, ed)
	}
	elapsed := time.Since(start)
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	debug.SetGCPercent(100)
	runtime.GC()
	return elapsed.Nanoseconds() / int64(n),
		(after.Mallocs - before.Mallocs) / uint64(n),
		(after.HeapAlloc - before.HeapAlloc) / uint64(n),
		len(v.AlertViolations)
}

func runCold(b *testing.B, name string, policy *storage.Policy, ed booleanpolicy.EnhancedDeployment) {
	matcher, err := booleanpolicy.BuildDeploymentMatcher(policy)
	require.NoError(b, err)
	ns, allocs, heap, viols := measure(matcher, ed, true)
	b.Logf("%-28s cold_cache: %8d ns/op  %6d allocs  %8d heap/op  %3d violations", name, ns, allocs, heap, viols)
}

// No runWarm for Bypass Cache — cache is always bypassed when filter is active.

func allAppImage() *storage.Image {
	img := benchImage(0, 50, 8)
	img.BaseImageInfo = []*storage.BaseImageInfo{{MaxLayerIndex: -1}}
	return img
}

func skipBasePolicy() *storage.Policy {
	return makePolicy(
		&storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_BASE},
		cvssGroup(),
	)
}

func skipAppPolicy() *storage.Policy {
	return makePolicy(
		&storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_APP},
		cvssGroup(),
	)
}

// Each benchmark is standalone. Run one at a time:
//   go test -bench BenchmarkCold_Baseline -v -run='^$' ./pkg/booleanpolicy/evalbench/
//   go test -bench BenchmarkWarm_100pct -v -run='^$' ./pkg/booleanpolicy/evalbench/

func BenchmarkCold_Baseline(b *testing.B) {
	runCold(b, "Baseline", makePolicy(nil, cvssGroup()), makeED(allAppImage()))
}
func BenchmarkCold_0pct(b *testing.B) {
	runCold(b, "0pct_removed", skipBasePolicy(), makeED(allAppImage()))
}
func BenchmarkCold_25pct(b *testing.B) {
	runCold(b, "25pct_removed", skipBasePolicy(), makeED(benchImage(1, 50, 8)))
}
func BenchmarkCold_50pct(b *testing.B) {
	runCold(b, "50pct_removed", skipBasePolicy(), makeED(benchImage(3, 50, 8)))
}
func BenchmarkCold_75pct(b *testing.B) {
	runCold(b, "75pct_removed", skipBasePolicy(), makeED(benchImage(5, 50, 8)))
}
func BenchmarkCold_100pct(b *testing.B) {
	runCold(b, "100pct_removed", skipAppPolicy(), makeED(allAppImage()))
}

// No BenchmarkWarm_* functions for Bypass Cache.
// Bypass Cache sets cache = nil internally when filter is active,
// so every call is effectively cold. Warm benchmarks would be
// misleading — they produce the same results as cold.
