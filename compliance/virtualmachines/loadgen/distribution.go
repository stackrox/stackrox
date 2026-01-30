package main

import (
	"math"
	"math/rand"
	"time"
)

// vmConfig holds the configuration for a single VM.
type vmConfig struct {
	cid            uint32
	numPackages    int
	reportInterval time.Duration
}

// assignVMConfigs assigns distribution-based configurations to VMs.
// Returns a slice of vmConfig, one per VM, with deterministic sampling based on seed.
func assignVMConfigs(vmCount int, startCID uint32, pkgDist packageDistribution, intervalDist intervalDistribution, seed int64) []vmConfig {
	rng := rand.New(rand.NewSource(seed))
	configs := make([]vmConfig, vmCount)

	for i := 0; i < vmCount; i++ {
		cid := startCID + uint32(i)

		// Sample package count from normal distribution
		pkgCount := sampleNormal(rng, pkgDist.Mean, pkgDist.Stddev, pkgDist.Min, pkgDist.Max)
		pkgCount = math.Round(pkgCount)
		if pkgCount < 1 {
			pkgCount = 1
		}

		// Sample interval from normal distribution (in seconds)
		intervalSeconds := sampleNormal(rng, intervalDist.Mean, intervalDist.Stddev, intervalDist.Min, intervalDist.Max)
		intervalDuration := time.Duration(math.Round(intervalSeconds)) * time.Second
		if intervalDuration < time.Second {
			intervalDuration = time.Second
		}

		configs[i] = vmConfig{
			cid:            cid,
			numPackages:    int(pkgCount),
			reportInterval: intervalDuration,
		}
	}

	return configs
}

// sampleNormal samples from a normal distribution and clamps to [min, max].
func sampleNormal(rng *rand.Rand, mean, stddev, min, max float64) float64 {
	// Generate standard normal variate
	z := rng.NormFloat64()
	// Transform to desired distribution
	value := mean + stddev*z
	// Clamp to bounds
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}


