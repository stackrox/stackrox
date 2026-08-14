package versioncompatibility

import (
	"sync/atomic"
	"testing"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version/internal"
	"github.com/stackrox/rox/pkg/version/productstreams"
)

func BenchmarkClassifyNoCache(b *testing.B) {
	internal.MainVersion = "4.5.0-testing"
	remote := productstreams.XYVersion{X: 4, Y: 3}

	for b.Loop() {
		xy, _ := productstreams.ParseXYFromVersionString("4.5.0-testing")
		versions, _ := makeCompatibleVersionRange(xy, AllowedSkew)
		classify(xy, versions, remote)
	}
}

func BenchmarkClassifyMutexSteadyState(b *testing.B) {
	internal.MainVersion = "4.5.0-testing"
	remote := productstreams.XYVersion{X: 4, Y: 3}

	var mu sync.Mutex
	var cached string
	var cXY productstreams.XYVersion
	var cRange []productstreams.XYVersion

	getMutex := func() (productstreams.XYVersion, []productstreams.XYVersion) {
		v := "4.5.0-testing"
		mu.Lock()
		defer mu.Unlock()
		if v != cached {
			cXY, _ = productstreams.ParseXYFromVersionString(v)
			cRange, _ = makeCompatibleVersionRange(cXY, AllowedSkew)
			cached = v
		}
		return cXY, cRange
	}

	for b.Loop() {
		xy, versions := getMutex()
		classify(xy, versions, remote)
	}
}

type cachedRange struct {
	mainVersion     string
	mainXY          productstreams.XYVersion
	compatibleRange []productstreams.XYVersion
}

func BenchmarkClassifyAtomicSteadyState(b *testing.B) {
	internal.MainVersion = "4.5.0-testing"
	remote := productstreams.XYVersion{X: 4, Y: 3}

	var cached atomic.Pointer[cachedRange]

	getAtomic := func() (productstreams.XYVersion, []productstreams.XYVersion) {
		current := "4.5.0-testing"
		if c := cached.Load(); c != nil && c.mainVersion == current {
			return c.mainXY, c.compatibleRange
		}
		xy, _ := productstreams.ParseXYFromVersionString(current)
		versions, _ := makeCompatibleVersionRange(xy, AllowedSkew)
		cached.Store(&cachedRange{mainVersion: current, mainXY: xy, compatibleRange: versions})
		return xy, versions
	}

	for b.Loop() {
		xy, versions := getAtomic()
		classify(xy, versions, remote)
	}
}

func BenchmarkClassifyNoCacheParallel(b *testing.B) {
	internal.MainVersion = "4.5.0-testing"
	remote := productstreams.XYVersion{X: 4, Y: 3}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			xy, _ := productstreams.ParseXYFromVersionString("4.5.0-testing")
			versions, _ := makeCompatibleVersionRange(xy, AllowedSkew)
			classify(xy, versions, remote)
		}
	})
}

func BenchmarkClassifyMutexSteadyStateParallel(b *testing.B) {
	internal.MainVersion = "4.5.0-testing"
	remote := productstreams.XYVersion{X: 4, Y: 3}

	var mu sync.Mutex
	var cached string
	var cXY productstreams.XYVersion
	var cRange []productstreams.XYVersion

	getMutex := func() (productstreams.XYVersion, []productstreams.XYVersion) {
		v := "4.5.0-testing"
		mu.Lock()
		defer mu.Unlock()
		if v != cached {
			cXY, _ = productstreams.ParseXYFromVersionString(v)
			cRange, _ = makeCompatibleVersionRange(cXY, AllowedSkew)
			cached = v
		}
		return cXY, cRange
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			xy, versions := getMutex()
			classify(xy, versions, remote)
		}
	})
}

func BenchmarkClassifyAtomicSteadyStateParallel(b *testing.B) {
	internal.MainVersion = "4.5.0-testing"
	remote := productstreams.XYVersion{X: 4, Y: 3}

	var cached atomic.Pointer[cachedRange]

	getAtomic := func() (productstreams.XYVersion, []productstreams.XYVersion) {
		current := "4.5.0-testing"
		if c := cached.Load(); c != nil && c.mainVersion == current {
			return c.mainXY, c.compatibleRange
		}
		xy, _ := productstreams.ParseXYFromVersionString(current)
		versions, _ := makeCompatibleVersionRange(xy, AllowedSkew)
		cached.Store(&cachedRange{mainVersion: current, mainXY: xy, compatibleRange: versions})
		return xy, versions
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			xy, versions := getAtomic()
			classify(xy, versions, remote)
		}
	})
}
