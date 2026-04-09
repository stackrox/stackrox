package internaltov2storage

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorProfileV2 converts internal api profiles to V2 storage profiles
func ComplianceOperatorProfileV2(internalMsg *central.ComplianceOperatorProfileV2, clusterID string) *storage.ComplianceOperatorProfileV2 {
	var rules []*storage.ComplianceOperatorProfileV2_Rule
	for _, r := range internalMsg.GetRules() {
		rules = append(rules, &storage.ComplianceOperatorProfileV2_Rule{
			RuleName: r.GetRuleName(),
		})
	}

	productType := internalMsg.GetAnnotations()[v1alpha1.ProductTypeAnnotation]

	operatorKind := centralToStorageProfileKind(internalMsg.GetOperatorKind())

	var equivalenceHash string
	if operatorKind == storage.ComplianceOperatorProfileV2_TAILORED_PROFILE {
		equivalenceHash = computeEquivalenceHash(internalMsg)
	}

	return &storage.ComplianceOperatorProfileV2{
		Id:              internalMsg.GetId(),
		ProfileId:       internalMsg.GetProfileId(),
		Name:            internalMsg.GetName(),
		ProfileVersion:  internalMsg.GetProfileVersion(),
		ProductType:     productType,
		Labels:          internalMsg.GetLabels(),
		Annotations:     internalMsg.GetAnnotations(),
		Description:     internalMsg.GetDescription(),
		Rules:           rules,
		Product:         internalMsg.GetAnnotations()[v1alpha1.ProductAnnotation],
		Title:           internalMsg.GetTitle(),
		Values:          internalMsg.GetValues(),
		ClusterId:       clusterID,
		ProfileRefId:    BuildProfileRefID(clusterID, internalMsg.GetProfileId(), productType),
		OperatorKind:    operatorKind,
		EquivalenceHash: equivalenceHash,
	}
}

// StorageToCentralProfileKind converts a storage profile OperatorKind to the internal API
// equivalent for sending to Sensor (e.g. when building scan config sync messages).
func StorageToCentralProfileKind(kind storage.ComplianceOperatorProfileV2_OperatorKind) central.ComplianceOperatorProfileV2_OperatorKind {
	switch kind {
	case storage.ComplianceOperatorProfileV2_PROFILE:
		return central.ComplianceOperatorProfileV2_PROFILE
	case storage.ComplianceOperatorProfileV2_TAILORED_PROFILE:
		return central.ComplianceOperatorProfileV2_TAILORED_PROFILE
	case storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED:
		// ROX-31229: Profiles stored by older Central versions may have UNSPECIFIED kind.
		// These are always regular profiles; treat as PROFILE. Mirrors centralToStorageProfileKind.
		// This fallback can be removed once Central versions that pre-date kind tracking are not supported.
		return central.ComplianceOperatorProfileV2_PROFILE
	default:
		return central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED
	}
}

// ProfileV2ToScanConfigRefs extracts name and kind from a slice of stored profiles into
// the scan configuration's ProfileReference storage type. Used when persisting profile_refs
// alongside a scan config so the startup sync path has correct kinds on reconnect.
func ProfileV2ToScanConfigRefs(profiles []*storage.ComplianceOperatorProfileV2) []*storage.ComplianceOperatorScanConfigurationV2_ProfileReference {
	refs := make([]*storage.ComplianceOperatorScanConfigurationV2_ProfileReference, 0, len(profiles))
	for _, p := range profiles {
		refs = append(refs, &storage.ComplianceOperatorScanConfigurationV2_ProfileReference{
			Name: p.GetName(),
			Kind: p.GetOperatorKind(),
		})
	}
	return refs
}

// ScanConfigRefsToCentral converts storage scan config ProfileReferences to the internal API
// equivalent for scan config messages sent to Sensor.
func ScanConfigRefsToCentral(refs []*storage.ComplianceOperatorScanConfigurationV2_ProfileReference) []*central.ApplyComplianceScanConfigRequest_BaseScanSettings_ProfileReference {
	centralRefs := make([]*central.ApplyComplianceScanConfigRequest_BaseScanSettings_ProfileReference, 0, len(refs))
	for _, ref := range refs {
		centralRefs = append(centralRefs, &central.ApplyComplianceScanConfigRequest_BaseScanSettings_ProfileReference{
			Name: ref.GetName(),
			Kind: StorageToCentralProfileKind(ref.GetKind()),
		})
	}
	return centralRefs
}

// computeEquivalenceHash returns a SHA-256 hex digest that identifies whether two
// tailored profiles carry equivalent compliance configuration. The hash covers the
// fields that define a profile's effective content: name, namespace, description,
// title, rules, and set_values.
//
// Rules and set_values are sorted for order independence. Rationale is excluded from
// set_values hashing because it is documentation, not configuration — two profiles
// with the same variable overrides but different rationales are functionally equivalent.
//
// IMPORTANT — changing the inputs or serialisation of this function changes the hash
// for every tailored profile. This may temporarily prevent multi-cluster scan config
// creation until all profiles are re-synced. Because the hash is computed in Central
// (not Sensor), rolling Sensor upgrades do NOT cause hash divergence.
func computeEquivalenceHash(msg *central.ComplianceOperatorProfileV2) string {
	h := sha256.New()

	// Write scalar fields, NUL-separated.
	for _, s := range []string{msg.GetName(), msg.GetNamespace(), msg.GetDescription(), msg.GetTitle()} {
		_, _ = h.Write([]byte(s))
		_, _ = h.Write([]byte{0})
	}

	// Rules: sorted for order independence.
	ruleNames := make([]string, 0, len(msg.GetRules()))
	for _, r := range msg.GetRules() {
		ruleNames = append(ruleNames, r.GetRuleName())
	}
	slices.Sort(ruleNames)
	for _, r := range ruleNames {
		_, _ = h.Write([]byte(r))
		_, _ = h.Write([]byte{0})
	}
	// Section separator between rules and set_values.
	_, _ = h.Write([]byte{0})

	// SetValues: sorted by name then value for order independence.
	type nameValue struct{ name, value string }
	setVals := make([]nameValue, 0, len(msg.GetSetValues()))
	for _, sv := range msg.GetSetValues() {
		setVals = append(setVals, nameValue{name: sv.GetName(), value: sv.GetValue()})
	}
	slices.SortFunc(setVals, func(a, b nameValue) int {
		if c := strings.Compare(a.name, b.name); c != 0 {
			return c
		}
		return strings.Compare(a.value, b.value)
	})
	for _, sv := range setVals {
		_, _ = h.Write([]byte(sv.name))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(sv.value))
		_, _ = h.Write([]byte{0})
	}

	return hex.EncodeToString(h.Sum(nil))
}

func centralToStorageProfileKind(kind central.ComplianceOperatorProfileV2_OperatorKind) storage.ComplianceOperatorProfileV2_OperatorKind {
	switch kind {
	case central.ComplianceOperatorProfileV2_PROFILE:
		return storage.ComplianceOperatorProfileV2_PROFILE
	case central.ComplianceOperatorProfileV2_TAILORED_PROFILE:
		return storage.ComplianceOperatorProfileV2_TAILORED_PROFILE
	case central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED:
		// ROX-31229: Older sensors do not set OperatorKind for regular (non-tailored)
		// profiles, so UNSPECIFIED is treated as PROFILE. This fallback can be
		// removed once versions that don't set OperatorKind (<= 4.10) are not supported.
		return storage.ComplianceOperatorProfileV2_PROFILE
	default:
		log.Warnf("Unexpected profile operator kind %v", kind)
		return storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED
	}
}
