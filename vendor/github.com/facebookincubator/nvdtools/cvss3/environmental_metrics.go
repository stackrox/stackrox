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

package cvss3

import "fmt"

type EnvironmentalMetrics struct {
	ConfidentialityRequirement
	IntegrityRequirement
	AvailabilityRequirement
	ModifiedAttackVector
	ModifiedAttackComplexity
	ModifiedPrivilegesRequired
	ModifiedUserInteraction
	ModifiedScope
	ModifiedConfidentiality
	ModifiedIntegrity
	ModifiedAvailability
}

type ConfidentialityRequirement int

const (
	ConfidentialityRequirementNotdefined ConfidentialityRequirement = iota
	ConfidentialityRequirementHigh
	ConfidentialityRequirementMedium
	ConfidentialityRequirementLow
)

var (
	weightsConfidentialityRequirement = []float64{1.0, 1.5, 1.0, 0.5}
	codeConfidentialityRequirement    = []string{"X", "H", "M", "L"}
)

func (cr ConfidentialityRequirement) defined() bool {
	return cr != ConfidentialityRequirementNotdefined
}

func (cr ConfidentialityRequirement) weight() float64 {
	return weightsConfidentialityRequirement[cr]
}

func (cr ConfidentialityRequirement) String() string {
	return codeConfidentialityRequirement[cr]
}

func (cr *ConfidentialityRequirement) parse(str string) error {
	idx, found := findIndex(str, codeConfidentialityRequirement)
	if found {
		*cr = ConfidentialityRequirement(idx)
		return nil
	}
	return fmt.Errorf("illegal confidentiality requirement code %s", str)
}

type IntegrityRequirement int

const (
	IntegrityRequirementNotdefined IntegrityRequirement = iota
	IntegrityRequirementHigh
	IntegrityRequirementMedium
	IntegrityRequirementLow
)

var (
	weightsIntegrityRequirement = []float64{1.0, 1.5, 1.0, 0.5}
	codeIntegrityRequirement    = []string{"X", "H", "M", "L"}
)

func (ir IntegrityRequirement) defined() bool {
	return ir != IntegrityRequirementNotdefined
}

func (ir IntegrityRequirement) weight() float64 {
	return weightsIntegrityRequirement[ir]
}

func (ir IntegrityRequirement) String() string {
	return codeIntegrityRequirement[ir]
}

func (ir *IntegrityRequirement) parse(str string) error {
	idx, found := findIndex(str, codeIntegrityRequirement)
	if found {
		*ir = IntegrityRequirement(idx)
		return nil
	}
	return fmt.Errorf("illegal integrity requirement code %s", str)
}

type AvailabilityRequirement int

const (
	AvailabilityRequirementNotdefined AvailabilityRequirement = iota
	AvailabilityRequirementHigh
	AvailabilityRequirementMedium
	AvailabilityRequirementLow
)

var (
	weightsAvailabilityRequirement = []float64{1.0, 1.5, 1.0, 0.5}
	codeAvailabilityRequirement    = []string{"X", "H", "M", "L"}
)

func (ar AvailabilityRequirement) defined() bool {
	return ar != AvailabilityRequirementNotdefined
}

func (ar AvailabilityRequirement) weight() float64 {
	return weightsAvailabilityRequirement[ar]
}

func (ar AvailabilityRequirement) String() string {
	return codeAvailabilityRequirement[ar]
}

func (ar *AvailabilityRequirement) parse(str string) error {
	idx, found := findIndex(str, codeAvailabilityRequirement)
	if found {
		*ar = AvailabilityRequirement(idx)
		return nil
	}
	return fmt.Errorf("illegal availability requirement code %s", str)
}

type ModifiedAttackVector AttackVector

const (
	AttackVectorNotdefined       ModifiedAttackVector = 0
	AttackVectorNotdefinedString string               = "X"
)

func (mav ModifiedAttackVector) defined() bool {
	return mav != AttackVectorNotdefined
}

func (mav ModifiedAttackVector) weight() float64 {
	if !mav.defined() {
		return 1.00
	}
	return AttackVector(mav).weight()
}

func (mav ModifiedAttackVector) String() string {
	if !mav.defined() {
		return AttackVectorNotdefinedString
	}
	return AttackVector(mav).String()
}

func (mav *ModifiedAttackVector) parse(str string) error {
	if str == AttackVectorNotdefinedString {
		*mav = AttackVectorNotdefined
		return nil
	}
	av := AttackVector(*mav)
	err := av.parse(str)
	*mav = ModifiedAttackVector(av)
	return err
}

type ModifiedAttackComplexity AttackComplexity

const (
	AttackComplexityNotdefined       ModifiedAttackComplexity = 0
	AttackComplexityNotdefinedString string                   = "X"
)

func (mac ModifiedAttackComplexity) defined() bool {
	return mac != AttackComplexityNotdefined
}

func (mac ModifiedAttackComplexity) weight() float64 {
	if !mac.defined() {
		return 1.00
	}
	return AttackComplexity(mac).weight()
}

func (mac ModifiedAttackComplexity) String() string {
	if !mac.defined() {
		return AttackComplexityNotdefinedString
	}
	return AttackComplexity(mac).String()
}

func (mac *ModifiedAttackComplexity) parse(str string) error {
	if str == AttackComplexityNotdefinedString {
		*mac = AttackComplexityNotdefined
		return nil
	}
	ac := AttackComplexity(*mac)
	err := ac.parse(str)
	*mac = ModifiedAttackComplexity(ac)
	return err
}

type ModifiedPrivilegesRequired PrivilegesRequired

const (
	PrivilegesRequiredNotdefined       ModifiedPrivilegesRequired = 0
	PrivilegesRequiredNotdefinedString string                     = "X"
)

func (mpr ModifiedPrivilegesRequired) defined() bool {
	return mpr != PrivilegesRequiredNotdefined
}

func (mpr ModifiedPrivilegesRequired) weight(scopeChanged bool) float64 {
	if !mpr.defined() {
		return 1.00
	}
	return PrivilegesRequired(mpr).weight(scopeChanged)
}

func (mpr ModifiedPrivilegesRequired) String() string {
	if !mpr.defined() {
		return PrivilegesRequiredNotdefinedString
	}
	return PrivilegesRequired(mpr).String()
}

func (mpr *ModifiedPrivilegesRequired) parse(str string) error {
	if str == PrivilegesRequiredNotdefinedString {
		*mpr = PrivilegesRequiredNotdefined
		return nil
	}
	pr := PrivilegesRequired(*mpr)
	err := pr.parse(str)
	*mpr = ModifiedPrivilegesRequired(pr)
	return err
}

type ModifiedUserInteraction UserInteraction

const (
	UserInteractionNotdefined       ModifiedUserInteraction = 0
	UserInteractionNotdefinedString string                  = "X"
)

func (mui ModifiedUserInteraction) defined() bool {
	return mui != UserInteractionNotdefined
}

func (mui ModifiedUserInteraction) weight() float64 {
	if !mui.defined() {
		return 1.00
	}
	return UserInteraction(mui).weight()
}

func (mui ModifiedUserInteraction) String() string {
	if !mui.defined() {
		return UserInteractionNotdefinedString
	}
	return UserInteraction(mui).String()
}

func (mui *ModifiedUserInteraction) parse(str string) error {
	if str == UserInteractionNotdefinedString {
		*mui = UserInteractionNotdefined
		return nil
	}
	ui := UserInteraction(*mui)
	err := ui.parse(str)
	*mui = ModifiedUserInteraction(ui)
	return err
}

type ModifiedScope Scope

const (
	ScopeNotdefined       ModifiedScope = 0
	ScopeNotdefinedString string        = "X"
)

func (ms ModifiedScope) defined() bool {
	return ms != ScopeNotdefined
}

func (ms ModifiedScope) String() string {
	if !ms.defined() {
		return ScopeNotdefinedString
	}
	return Scope(ms).String()
}

func (ms *ModifiedScope) parse(str string) error {
	if str == ScopeNotdefinedString {
		*ms = ScopeNotdefined
		return nil
	}
	s := Scope(*ms)
	err := s.parse(str)
	*ms = ModifiedScope(s)
	return err
}

type ModifiedConfidentiality Confidentiality

const (
	ConfidentialityNotdefined       ModifiedConfidentiality = 0
	ConfidentialityNotdefinedString string                  = "X"
)

func (mc ModifiedConfidentiality) defined() bool {
	return mc != ConfidentialityNotdefined
}

func (mc ModifiedConfidentiality) weight() float64 {
	if !mc.defined() {
		return 1.00
	}
	return Confidentiality(mc).weight()
}

func (mc ModifiedConfidentiality) String() string {
	if !mc.defined() {
		return ConfidentialityNotdefinedString
	}
	return Confidentiality(mc).String()
}

func (mc *ModifiedConfidentiality) parse(str string) error {
	if str == ConfidentialityNotdefinedString {
		*mc = ConfidentialityNotdefined
		return nil
	}
	c := Confidentiality(*mc)
	err := c.parse(str)
	*mc = ModifiedConfidentiality(c)
	return err
}

type ModifiedIntegrity Integrity

const (
	IntegrityNotdefined       ModifiedIntegrity = 0
	IntegrityNotdefinedString string            = "X"
)

func (mi ModifiedIntegrity) defined() bool {
	return mi != IntegrityNotdefined
}

func (mi ModifiedIntegrity) weight() float64 {
	if !mi.defined() {
		return 1.00
	}
	return Integrity(mi).weight()
}

func (mi ModifiedIntegrity) String() string {
	if !mi.defined() {
		return IntegrityNotdefinedString
	}
	return Integrity(mi).String()
}

func (mi *ModifiedIntegrity) parse(str string) error {
	if str == IntegrityNotdefinedString {
		*mi = IntegrityNotdefined
		return nil
	}
	i := Integrity(*mi)
	err := i.parse(str)
	*mi = ModifiedIntegrity(i)
	return err
}

type ModifiedAvailability Availability

const (
	AvailabilityNotdefined       ModifiedAvailability = 0
	AvailabilityNotdefinedString string               = "X"
)

func (ma ModifiedAvailability) defined() bool {
	return ma != AvailabilityNotdefined
}

func (ma ModifiedAvailability) weight() float64 {
	if !ma.defined() {
		return 1.00
	}
	return Availability(ma).weight()
}

func (ma ModifiedAvailability) String() string {
	if !ma.defined() {
		return AvailabilityNotdefinedString
	}
	return Availability(ma).String()
}

func (ma *ModifiedAvailability) parse(str string) error {
	if str == AvailabilityNotdefinedString {
		*ma = AvailabilityNotdefined
		return nil
	}
	a := Availability(*ma)
	err := a.parse(str)
	*ma = ModifiedAvailability(a)
	return err
}
