// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cvss2

import (
	"fmt"
	"math"
)

func roundTo1Decimal(x float64) float64 {
	return math.Round(x*10) / 10
}

// Validate should be called before calculating any scores on vector
// If there's an error, there's no guarantee that a call to *Score() won't panic
func (v Vector) Validate() error {
	switch {
	case !v.BaseMetrics.AccessVector.defined():
		return fmt.Errorf("base metric access vector not defined")
	case !v.BaseMetrics.AccessComplexity.defined():
		return fmt.Errorf("base metric access complexity not defined")
	case !v.BaseMetrics.Authentication.defined():
		return fmt.Errorf("base metric authentication not defined")
	case !v.BaseMetrics.ConfidentialityImpact.defined():
		return fmt.Errorf("base metric confidentiality impact not defined")
	case !v.BaseMetrics.IntegrityImpact.defined():
		return fmt.Errorf("base metric integrity impact not defined")
	case !v.BaseMetrics.AvailabilityImpact.defined():
		return fmt.Errorf("base metric availability impact not defined")
	default:
		return nil
	}
}

// Score = combined score for the whole Vector
func (v Vector) Score() float64 {
	// combines all of them
	return v.EnvironmentalScore()
}

// BaseScore returns base score of the vector
func (v Vector) BaseScore() float64 {
	return v.baseScoreWith(v.ImpactScore(false))
}

// TemporalScore returns temporal score of the vector
func (v Vector) TemporalScore() float64 {
	return v.temporalScoreWith(v.ImpactScore(false))
}

// EnvironmentalScore returns environmental score of the vector
func (v Vector) EnvironmentalScore() float64 {
	ai := math.Min(10, v.ImpactScore(true))
	at := v.temporalScoreWith(ai)
	return roundTo1Decimal((at + (10-at)*v.EnvironmentalMetrics.CollateralDamagePotential.weight()) * v.EnvironmentalMetrics.TargetDistribution.weight())
}

// helpers

// ImpactScore returns impact score of the vector
func (v Vector) ImpactScore(adjust bool) float64 {
	c := v.BaseMetrics.ConfidentialityImpact.weight()
	i := v.BaseMetrics.IntegrityImpact.weight()
	a := v.BaseMetrics.AvailabilityImpact.weight()

	if adjust {
		c *= v.EnvironmentalMetrics.ConfidentialityRequirement.weight()
		i *= v.EnvironmentalMetrics.IntegrityRequirement.weight()
		a *= v.EnvironmentalMetrics.AvailabilityRequirement.weight()
	}

	return 10.41 * (1 - (1-c)*(1-i)*(1-a))
}

// ExploitabilityScore returns exploitability score of the vector
func (v Vector) ExploitabilityScore() float64 {
	return 20 * v.BaseMetrics.AccessVector.weight() * v.BaseMetrics.AccessComplexity.weight() * v.BaseMetrics.Authentication.weight()
}

func (v Vector) temporalScoreWith(impact float64) float64 {
	return roundTo1Decimal(v.baseScoreWith(impact) *
		v.TemporalMetrics.Exploitablity.weight() *
		v.TemporalMetrics.RemediationLevel.weight() *
		v.TemporalMetrics.ReportConfidence.weight())
}

func (v Vector) baseScoreWith(impact float64) float64 {
	if impact == 0.0 {
		return 0.0
	}
	return roundTo1Decimal((0.6*impact + 0.4*v.ExploitabilityScore() - 1.5) * 1.176)
}
