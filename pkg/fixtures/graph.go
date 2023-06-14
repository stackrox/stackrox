package fixtures

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/nodes/converter"
)

///////////////////////////////
// connected datastore synthetic dataset //
///////////////////////////////

/////////////////////
// Image with CVEs //

// Data relationships
//
//     Namespace (scope information holder)
//         ^ 1
//         |
//         v *
//     Deployment
//         ^ *
//         |
//         v *
//       Image <--------------+ *
//         ^ *                |
//         |                  |
// ImageComponentEdge         |
//         |                  |
//         v *                |
//   ImageComponent      ImageCVEEdge
//         ^ *                |
//         |                  |
//  ComponentCVEEdge          |
//         |                  |
//         v *                |
//        CVE <---------------+ *
//
// Three types of data have to be injected here:
// - NamespaceMetadata
// - Deployment
// - Image
//
// * NamespaceMetadata is used to provide the scope information
//
// * The Deployment links namespace (scope) with deployed images (possibly using multiple containers
// with one image per container. The reference to the image contains the image ID (sha image hash),
// as well as image name information.
//
// * The image information is injected in the form of an image with scan information.
// ** The Image Scan field contains the data that will be used to fill the connected datastore storage
// ** The Image Scan contains EmbeddedImageScanComponent objects, which are used to populate
// ImageComponent as well as ImageComponentEdge storage entities.
// ** EmbeddedImageScanComponent contains in turn EmbeddedVulnerability objects which contain CVE data
// and are used to populate the CVE, ComponentCVEEdge and ImageCVEEdge storage entities.
//
//
// For testing purposes, a graph of objects like the one below could be used.
//
// Cluster1 -- NamespaceA -- Deployment1 -- Image1 --+--> ImageComponent1 --+--> ImageCVE1
//                                                   |                      |
//                                                   |                      +--------+
//                                                   |                               |
//                                                   |                               v
//                                                   |                           ImageCVE2
//                                                   |                               ^
//                                                   |                               |
//                                                   |                               +--------+
//                                                   |                                        |
//                                                   +--> ImageComponent2 -----> ImageCVE3    |
//                                                   |                                        |
//                                                   +-----------+                            |
//                                                               |          +--> ImageCVE4    |
//                                                               v          |                 |
//                                                        ImageComponent3 --+                 |
//                                                               ^          |                 |
//                                                               |          +--> ImageCVE5    |
//                                                   +-----------+                            |
//                                                   |                                        |
//                                                   +--> ImageComponent4                     |
//                                                   |                                        |
//                                                   |                      +-----------------+
//                                                   |                      |
// Cluster2 -- NamespaceB -- Deployment2 -- Image2 --+--> ImageComponent5 --+--> ImageCVE6
//                                                                          |
//                                                                          +--> ImageCVE7

// GetEmbeddedImageCVE1234x0001 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE1234x0001() *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:          "CVE-1234-0001",
		Cvss:         5.8,
		Summary:      "Find some inspiring quote on an evil topic to insert here.",
		Link:         "book://author/title",
		SetFixedBy:   &storage.EmbeddedVulnerability_FixedBy{FixedBy: ""},
		ScoreVersion: storage.EmbeddedVulnerability_V2,
		CvssV2: &storage.CVSSV2{
			Vector:              "AV:N/AC:M/Au:N/C:P/I:P/A:N",
			AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
			AccessComplexity:    storage.CVSSV2_ACCESS_MEDIUM,
			Authentication:      storage.CVSSV2_AUTH_NONE,
			Confidentiality:     storage.CVSSV2_IMPACT_PARTIAL,
			Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
			Availability:        storage.CVSSV2_IMPACT_NONE,
			ExploitabilityScore: 8.6,
			ImpactScore:         4.9,
			Score:               5.8,
			Severity:            storage.CVSSV2_MEDIUM,
		},
		CvssV3:            nil,
		PublishedOn:       &types.Timestamp{Seconds: 1234567890},
		LastModified:      &types.Timestamp{Seconds: 1235467890},
		VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{
			storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		},
		Suppressed:            false,
		SuppressActivation:    nil,
		SuppressExpiry:        nil,
		FirstSystemOccurrence: &types.Timestamp{Seconds: 1243567890},
		FirstImageOccurrence:  &types.Timestamp{Seconds: 1245367890},
		Severity:              storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		State:                 storage.VulnerabilityState_OBSERVED,
	}
}

// GetEmbeddedImageCVE4567x0002 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE4567x0002() *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:          "CVE-4567-0002",
		Cvss:         7.5,
		Summary:      "Find some inspiring quote on an evil topic to insert here.",
		Link:         "book://author/title",
		SetFixedBy:   &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.1.1"},
		ScoreVersion: storage.EmbeddedVulnerability_V3,
		CvssV2: &storage.CVSSV2{
			Vector:              "AV:N/AC:L/Au:N/C:N/I:P/A:N",
			AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
			AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
			Authentication:      storage.CVSSV2_AUTH_NONE,
			Confidentiality:     storage.CVSSV2_IMPACT_NONE,
			Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
			Availability:        storage.CVSSV2_IMPACT_NONE,
			ExploitabilityScore: 10.0,
			ImpactScore:         2.9,
			Score:               5.0,
			Severity:            storage.CVSSV2_MEDIUM,
		},
		CvssV3: &storage.CVSSV3{
			Vector:              "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
			ExploitabilityScore: 3.9,
			ImpactScore:         3.6,
			AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
			AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
			PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
			UserInteraction:     storage.CVSSV3_UI_NONE,
			Scope:               storage.CVSSV3_UNCHANGED,
			Confidentiality:     storage.CVSSV3_IMPACT_NONE,
			Integrity:           storage.CVSSV3_IMPACT_HIGH,
			Availability:        storage.CVSSV3_IMPACT_NONE,
			Score:               7.5,
			Severity:            storage.CVSSV3_HIGH,
		},
		PublishedOn:       &types.Timestamp{Seconds: 1234567890},
		LastModified:      &types.Timestamp{Seconds: 1235467890},
		VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{
			storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		},
		Suppressed:            false,
		SuppressActivation:    nil,
		SuppressExpiry:        nil,
		FirstSystemOccurrence: &types.Timestamp{Seconds: 1243567890},
		FirstImageOccurrence:  &types.Timestamp{Seconds: 1245367890},
		Severity:              storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		State:                 storage.VulnerabilityState_OBSERVED,
	}
}

// GetEmbeddedImageCVE1234x0003 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE1234x0003() *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:          "CVE-1234-0003",
		Cvss:         7.5,
		Summary:      "Find some inspiring quote on an evil topic to insert here.",
		Link:         "book://author/title",
		SetFixedBy:   &storage.EmbeddedVulnerability_FixedBy{FixedBy: ""},
		ScoreVersion: storage.EmbeddedVulnerability_V3,
		CvssV2: &storage.CVSSV2{
			Vector:              "AV:N/AC:L/Au:N/C:N/I:N/A:P",
			AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
			AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
			Authentication:      storage.CVSSV2_AUTH_NONE,
			Confidentiality:     storage.CVSSV2_IMPACT_NONE,
			Integrity:           storage.CVSSV2_IMPACT_NONE,
			Availability:        storage.CVSSV2_IMPACT_PARTIAL,
			ExploitabilityScore: 10.0,
			ImpactScore:         2.9,
			Score:               5.0,
			Severity:            storage.CVSSV2_MEDIUM,
		},
		CvssV3: &storage.CVSSV3{
			Vector:              "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
			ExploitabilityScore: 3.9,
			ImpactScore:         3.6,
			AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
			AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
			PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
			UserInteraction:     storage.CVSSV3_UI_NONE,
			Scope:               storage.CVSSV3_UNCHANGED,
			Confidentiality:     storage.CVSSV3_IMPACT_NONE,
			Integrity:           storage.CVSSV3_IMPACT_NONE,
			Availability:        storage.CVSSV3_IMPACT_HIGH,
			Score:               7.5,
			Severity:            storage.CVSSV3_HIGH,
		},
		PublishedOn:       &types.Timestamp{Seconds: 1234567890},
		LastModified:      &types.Timestamp{Seconds: 1235467890},
		VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{
			storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		},
		Suppressed:            false,
		SuppressActivation:    nil,
		SuppressExpiry:        nil,
		FirstSystemOccurrence: &types.Timestamp{Seconds: 1243567890},
		FirstImageOccurrence:  &types.Timestamp{Seconds: 1245367890},
		Severity:              storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		State:                 storage.VulnerabilityState_OBSERVED,
	}
}

// GetEmbeddedImageCVE3456x0004 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE3456x0004() *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:          "CVE-3456-0004",
		Cvss:         7.5,
		Summary:      "Find some inspiring quote on an evil topic to insert here.",
		Link:         "book://author/title",
		SetFixedBy:   &storage.EmbeddedVulnerability_FixedBy{FixedBy: ""},
		ScoreVersion: storage.EmbeddedVulnerability_V3,
		CvssV2: &storage.CVSSV2{
			Vector:              "AV:N/AC:M/Au:N/C:N/I:N/A:P",
			AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
			AccessComplexity:    storage.CVSSV2_ACCESS_MEDIUM,
			Authentication:      storage.CVSSV2_AUTH_NONE,
			Confidentiality:     storage.CVSSV2_IMPACT_NONE,
			Integrity:           storage.CVSSV2_IMPACT_NONE,
			Availability:        storage.CVSSV2_IMPACT_PARTIAL,
			ExploitabilityScore: 8.6,
			ImpactScore:         2.9,
			Score:               4.3,
			Severity:            storage.CVSSV2_MEDIUM,
		},
		CvssV3: &storage.CVSSV3{
			Vector:              "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H",
			ExploitabilityScore: 2.2,
			ImpactScore:         3.6,
			AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
			AttackComplexity:    storage.CVSSV3_COMPLEXITY_HIGH,
			PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
			UserInteraction:     storage.CVSSV3_UI_NONE,
			Scope:               storage.CVSSV3_UNCHANGED,
			Confidentiality:     storage.CVSSV3_IMPACT_NONE,
			Integrity:           storage.CVSSV3_IMPACT_NONE,
			Availability:        storage.CVSSV3_IMPACT_HIGH,
			Score:               5.9,
			Severity:            storage.CVSSV3_MEDIUM,
		},
		PublishedOn:       &types.Timestamp{Seconds: 1234567890},
		LastModified:      &types.Timestamp{Seconds: 1235467890},
		VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{
			storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		},
		Suppressed:            false,
		SuppressActivation:    nil,
		SuppressExpiry:        nil,
		FirstSystemOccurrence: &types.Timestamp{Seconds: 1243567890},
		FirstImageOccurrence:  &types.Timestamp{Seconds: 1245367890},
		Severity:              storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		State:                 storage.VulnerabilityState_OBSERVED,
	}
}

// GetEmbeddedImageCVE3456x0005 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE3456x0005() *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:          "CVE-3456-0005",
		Cvss:         5.3,
		Summary:      "Find some inspiring quote on an evil topic to insert here.",
		Link:         "book://author/title",
		SetFixedBy:   &storage.EmbeddedVulnerability_FixedBy{FixedBy: ""},
		ScoreVersion: storage.EmbeddedVulnerability_V3,
		CvssV2: &storage.CVSSV2{
			Vector:              "AV:L/AC:L/Au:N/C:P/I:P/A:P",
			AttackVector:        storage.CVSSV2_ATTACK_LOCAL,
			AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
			Authentication:      storage.CVSSV2_AUTH_NONE,
			Confidentiality:     storage.CVSSV2_IMPACT_PARTIAL,
			Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
			Availability:        storage.CVSSV2_IMPACT_PARTIAL,
			ExploitabilityScore: 3.9,
			ImpactScore:         6.4,
			Score:               4.6,
			Severity:            storage.CVSSV2_MEDIUM,
		},
		CvssV3: &storage.CVSSV3{
			Vector:              "CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:L/A:L",
			ExploitabilityScore: 1.8,
			ImpactScore:         3.4,
			AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
			AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
			PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
			UserInteraction:     storage.CVSSV3_UI_NONE,
			Scope:               storage.CVSSV3_UNCHANGED,
			Confidentiality:     storage.CVSSV3_IMPACT_LOW,
			Integrity:           storage.CVSSV3_IMPACT_LOW,
			Availability:        storage.CVSSV3_IMPACT_LOW,
			Score:               5.3,
			Severity:            storage.CVSSV3_MEDIUM,
		},
		PublishedOn:       &types.Timestamp{Seconds: 1234567890},
		LastModified:      &types.Timestamp{Seconds: 1235467890},
		VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{
			storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		},
		Suppressed:            false,
		SuppressActivation:    nil,
		SuppressExpiry:        nil,
		FirstSystemOccurrence: &types.Timestamp{Seconds: 1243567890},
		FirstImageOccurrence:  &types.Timestamp{Seconds: 1245367890},
		Severity:              storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		State:                 storage.VulnerabilityState_OBSERVED,
	}
}

// GetEmbeddedImageCVE2345x0006 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE2345x0006() *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:          "CVE-2345-0006",
		Cvss:         7.8,
		Summary:      "Find some inspiring quote on an evil topic to insert here.",
		Link:         "book://author/title",
		SetFixedBy:   &storage.EmbeddedVulnerability_FixedBy{FixedBy: ""},
		ScoreVersion: storage.EmbeddedVulnerability_V3,
		CvssV2: &storage.CVSSV2{
			Vector:              "AV:N/AC:M/Au:N/C:P/I:P/A:P",
			AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
			AccessComplexity:    storage.CVSSV2_ACCESS_MEDIUM,
			Authentication:      storage.CVSSV2_AUTH_NONE,
			Confidentiality:     storage.CVSSV2_IMPACT_PARTIAL,
			Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
			Availability:        storage.CVSSV2_IMPACT_PARTIAL,
			ExploitabilityScore: 8.6,
			ImpactScore:         6.4,
			Score:               6.8,
			Severity:            storage.CVSSV2_MEDIUM,
		},
		CvssV3: &storage.CVSSV3{
			Vector:              "CVSS:3.0/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
			ExploitabilityScore: 1.8,
			ImpactScore:         5.9,
			AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
			AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
			PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
			UserInteraction:     storage.CVSSV3_UI_REQUIRED,
			Scope:               storage.CVSSV3_UNCHANGED,
			Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
			Integrity:           storage.CVSSV3_IMPACT_HIGH,
			Availability:        storage.CVSSV3_IMPACT_HIGH,
			Score:               7.8,
			Severity:            storage.CVSSV3_HIGH,
		},
		PublishedOn:       &types.Timestamp{Seconds: 1234567890},
		LastModified:      &types.Timestamp{Seconds: 1235467890},
		VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{
			storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		},
		Suppressed:            false,
		SuppressActivation:    nil,
		SuppressExpiry:        nil,
		FirstSystemOccurrence: &types.Timestamp{Seconds: 1243567890},
		FirstImageOccurrence:  &types.Timestamp{Seconds: 1245367890},
		Severity:              storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		State:                 storage.VulnerabilityState_OBSERVED,
	}
}

// GetEmbeddedImageCVE2345x0007 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE2345x0007() *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:          "CVE-2345-0007",
		Cvss:         5.9,
		Summary:      "Find some inspiring quote on an evil topic to insert here.",
		Link:         "book://author/title",
		SetFixedBy:   &storage.EmbeddedVulnerability_FixedBy{FixedBy: "2.5.6"},
		ScoreVersion: storage.EmbeddedVulnerability_V3,
		CvssV2: &storage.CVSSV2{
			Vector:              "AV:N/AC:M/Au:N/C:N/I:P/A:N",
			AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
			AccessComplexity:    storage.CVSSV2_ACCESS_MEDIUM,
			Authentication:      storage.CVSSV2_AUTH_NONE,
			Confidentiality:     storage.CVSSV2_IMPACT_NONE,
			Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
			Availability:        storage.CVSSV2_IMPACT_NONE,
			ExploitabilityScore: 8.6,
			ImpactScore:         2.9,
			Score:               4.3,
			Severity:            storage.CVSSV2_MEDIUM,
		},
		CvssV3: &storage.CVSSV3{
			Vector:              "CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N",
			ExploitabilityScore: 2.2,
			ImpactScore:         3.6,
			AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
			AttackComplexity:    storage.CVSSV3_COMPLEXITY_HIGH,
			PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
			UserInteraction:     storage.CVSSV3_UI_NONE,
			Scope:               storage.CVSSV3_UNCHANGED,
			Confidentiality:     storage.CVSSV3_IMPACT_NONE,
			Integrity:           storage.CVSSV3_IMPACT_HIGH,
			Availability:        storage.CVSSV3_IMPACT_NONE,
			Score:               5.9,
			Severity:            storage.CVSSV3_MEDIUM,
		},
		PublishedOn:       &types.Timestamp{Seconds: 1234567890},
		LastModified:      &types.Timestamp{Seconds: 1235467890},
		VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{
			storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		},
		Suppressed:            false,
		SuppressActivation:    nil,
		SuppressExpiry:        nil,
		FirstSystemOccurrence: &types.Timestamp{Seconds: 1243567890},
		FirstImageOccurrence:  &types.Timestamp{Seconds: 1245367890},
		Severity:              storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		State:                 storage.VulnerabilityState_OBSERVED,
	}
}

// GetEmbeddedImageComponent1x1 provides a pseudo-realistic image component for connected datastore integration testing.
func GetEmbeddedImageComponent1x1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "scarlet",
		Version: "1.1",
		License: nil,
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE1234x0001(),
			GetEmbeddedImageCVE4567x0002(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		Priority:      0,
		Source:        storage.SourceType_OS,
		Location:      "",
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 7.5},
		RiskScore:     2.154,
		FixedBy:       "1.1.1",
		Executables:   []*storage.EmbeddedImageScanComponent_Executable{},
	}
}

// GetEmbeddedImageComponent1x2 provides a pseudo-realistic image component for connected datastore integration testing.
func GetEmbeddedImageComponent1x2() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "baskerville",
		Version: "1.2",
		License: nil,
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE1234x0003(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 1},
		Priority:      0,
		Source:        storage.SourceType_PYTHON,
		Location:      "",
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 7.5},
		RiskScore:     1.1625,
		FixedBy:       "1.2.5",
		Executables: []*storage.EmbeddedImageScanComponent_Executable{
			{
				Path: "/horrific/hound",
			},
		},
	}
}

// GetEmbeddedImageComponent1s2x3 provides a pseudo-realistic image component for connected datastore integration testing.
func GetEmbeddedImageComponent1s2x3() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "downtown-london",
		Version: "1s.2-3",
		License: nil,
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE3456x0004(),
			GetEmbeddedImageCVE3456x0005(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 1},
		Priority:      0,
		Source:        storage.SourceType_JAVA,
		Location:      "",
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 7.5},
		RiskScore:     1.1625,
		FixedBy:       "",
		Executables:   []*storage.EmbeddedImageScanComponent_Executable{},
	}
}

// GetEmbeddedImageComponent2x4 provides a pseudo-realistic image component for connected datastore integration testing.
func GetEmbeddedImageComponent2x4() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "dr-jekyll-medecine-practice",
		Version:       "2.4",
		License:       nil,
		Vulns:         []*storage.EmbeddedVulnerability{},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		Priority:      0,
		Source:        storage.SourceType_INFRASTRUCTURE,
		Location:      "",
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 0.0},
		RiskScore:     0.0,
		FixedBy:       "",
		Executables:   []*storage.EmbeddedImageScanComponent_Executable{},
	}
}

// GetEmbeddedImageComponent2x5 provides a pseudo-realistic image component for connected datastore integration testing.
func GetEmbeddedImageComponent2x5() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "mr-hyde-secret-entrance",
		Version: "2.5",
		License: nil,
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE4567x0002(),
			GetEmbeddedImageCVE2345x0006(),
			GetEmbeddedImageCVE2345x0007(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 2},
		Priority:      0,
		Source:        storage.SourceType_RUBY,
		Location:      "",
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 7.8},
		RiskScore:     0.0,
		FixedBy:       "2.5.6",
		Executables: []*storage.EmbeddedImageScanComponent_Executable{
			{
				Path: "/murderous/cane",
				Dependencies: []string{
					"experimental-powder",
				},
			},
		},
	}
}

// GetImageSherlockHolmes1 provides a pseudo-realistic image for connected datastore integration testing.
func GetImageSherlockHolmes1() *storage.Image {
	return &storage.Image{
		Id: "sha256:50fa59cca653c51d194974830826ff7a9d9095175f78caf40d5423d3fb12c4f7",
		Name: &storage.ImageName{
			Registry: "baker.st",
			Remote:   "sherlock/holmes",
			Tag:      "v1",
			FullName: "baker.st/sherlock/holmes:v1",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Digest:  "sha256:0a488a3872bfcd9e79a3575b5c273b01c01a21b16e86213a26eb7f3ab540eb84",
				Created: &types.Timestamp{Seconds: 1553642092, Nanos: 227945051},
				Author:  "Sir Arthur Conan Doyle",
				Layers: []*storage.ImageLayer{
					{
						Instruction: "COPY",
						Value:       "/ / # buildkit",
						Created:     &types.Timestamp{Seconds: 1553640086, Nanos: 106246179},
					},
					{
						Instruction: " /usr/local/bin/ # buildkit",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     &types.Timestamp{Seconds: 1553640126, Nanos: 263243615},
					},
					{
						Instruction: "ADD",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     &types.Timestamp{Seconds: 1553640134, Nanos: 213199897},
					},
				},
				User:       "root",
				Command:    nil,
				Entrypoint: nil,
				Volumes:    nil,
				Labels:     nil,
			},
			V2: &storage.V2Metadata{Digest: "sha256:4d818f38fa9dcbf41e7c255f276a72e5c471c1523b6f755a344bac04652351dd"},
			LayerShas: []string{
				"sha256:50fa59cca653c51d194974830826ff7a9d9095175f78caf40d5423d3fb12c4f7",
				"sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				"sha256:8e2ee98ae01ebe81fe221f5a444cab18c8f7a26cd00ce1ef23cf7432feef99b4",
			},
			DataSource: &storage.DataSource{
				Id:   "13e92196-8216-4714-9ac7-fac779bb973b",
				Name: "Sir Arthur Conan Doyle",
			},
			Version: 0,
		},
		Scan: &storage.ImageScan{
			ScannerVersion: "2.24.0-11-g05cf175999",
			ScanTime:       &types.Timestamp{Seconds: 1654154310, Nanos: 970783800},
			Components: []*storage.EmbeddedImageScanComponent{
				GetEmbeddedImageComponent1x1(),
				GetEmbeddedImageComponent1x2(),
				GetEmbeddedImageComponent1s2x3(),
			},
			OperatingSystem: "crime-stories",
			DataSource: &storage.DataSource{
				Id:   "169b0d3f-8277-4900-bbce-1127077defae",
				Name: "Stackrox Scanner",
			},
			Notes: []storage.ImageScan_Note{},
		},
		SignatureVerificationData: nil,
		Signature:                 nil,
		SetComponents:             &storage.Image_Components{Components: 3},
		SetCves:                   &storage.Image_Cves{Cves: 5},
		SetFixable:                &storage.Image_FixableCves{FixableCves: 2},
		LastUpdated:               &types.Timestamp{Seconds: 1654154313, Nanos: 67882700},
		NotPullable:               false,
		IsClusterLocal:            false,
		Priority:                  0,
		RiskScore:                 1.5,
		SetTopCvss:                &storage.Image_TopCvss{TopCvss: 7.5},
		Notes: []storage.Image_Note{
			storage.Image_MISSING_SIGNATURE_VERIFICATION_DATA,
			storage.Image_MISSING_SIGNATURE,
		},
	}
}

// GetImageDoctorJekyll2 provides a pseudo-realistic image for connected datastore integration testing.
func GetImageDoctorJekyll2() *storage.Image {
	return &storage.Image{
		Id: "sha256:835762dc5388a591ecf31540eaeb14ec8bc96ad48a3bd11fdef77b7106111eec",
		Name: &storage.ImageName{
			Registry: "book.worm",
			Remote:   "doctor/jekyll",
			Tag:      "v2",
			FullName: "book.worm/doctor/jekyll:v2",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Digest:  "sha256:9fe0366ee2eead5a66948f853ebedae5464361b5ffb166980db355d294a971ff",
				Created: &types.Timestamp{Seconds: 1553642392, Nanos: 877872600},
				Author:  "Sir Arthur Conan Doyle",
				Layers: []*storage.ImageLayer{
					{
						Instruction: "COPY",
						Value:       "/ / # buildkit",
						Created:     &types.Timestamp{Seconds: 1553641386, Nanos: 227945051},
					},
					{
						Instruction: " /usr/local/bin/ # buildkit",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     &types.Timestamp{Seconds: 1553641426, Nanos: 106246179},
					},
					{
						Instruction: "ADD",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     &types.Timestamp{Seconds: 1553641534, Nanos: 302497847},
					},
				},
				User:       "root",
				Command:    nil,
				Entrypoint: nil,
				Volumes:    nil,
				Labels:     nil,
			},
			V2: &storage.V2Metadata{Digest: "sha256:1e0ccd4630c681f887d799677a5d846d13e7fb69d4c4e25b899ba12ce804ac06"},
			LayerShas: []string{
				"sha256:8d5041d30882e1fff9d4f4f90a72bd22896c8e365ad6095189d70d605bf7d3bd",
				"sha256:9bc15005c6e7e93dcbe4c05f61dae53fcd72ce81aa44b3ae3d989e910ac682b9",
				"sha256:e83e783ca7567afba7ea4541e6233032ca0303b43811539453c36b11c497eda8",
			},
			DataSource: &storage.DataSource{
				Id:   "28eacb99-4e61-8be8-c316e6875184",
				Name: "Robert Louis Stevenson",
			},
			Version: 0,
		},
		Scan: &storage.ImageScan{
			ScannerVersion: "2.24.0-11-g05cf175999",
			ScanTime:       &types.Timestamp{Seconds: 1654154710, Nanos: 67882700},
			Components: []*storage.EmbeddedImageScanComponent{
				GetEmbeddedImageComponent1s2x3(),
				GetEmbeddedImageComponent2x4(),
				GetEmbeddedImageComponent2x5(),
			},
			OperatingSystem: "crime-stories",
			DataSource: &storage.DataSource{
				Id:   "169b0d3f-8277-4900-bbce-1127077defae",
				Name: "Stackrox Scanner",
			},
			Notes: []storage.ImageScan_Note{},
		},
		SignatureVerificationData: nil,
		Signature:                 nil,
		SetComponents:             &storage.Image_Components{Components: 3},
		SetCves:                   &storage.Image_Cves{Cves: 5},
		SetFixable:                &storage.Image_FixableCves{FixableCves: 2},
		LastUpdated:               &types.Timestamp{Seconds: 1654154413, Nanos: 970783800},
		NotPullable:               false,
		IsClusterLocal:            false,
		Priority:                  0,
		RiskScore:                 2.375,
		SetTopCvss:                &storage.Image_TopCvss{TopCvss: 7.8},
		Notes: []storage.Image_Note{
			storage.Image_MISSING_SIGNATURE_VERIFICATION_DATA,
			storage.Image_MISSING_SIGNATURE,
		},
	}
}

// GetDeploymentSherlockHolmes1 provides a pseudo-realistic deployment for connected datastore integration testing.
func GetDeploymentSherlockHolmes1(id string, namespace *storage.NamespaceMetadata) *storage.Deployment {
	return &storage.Deployment{
		Id:                    id,
		Name:                  "sherlock-holmes-deployment",
		Hash:                  0,
		Type:                  "Deployment",
		Namespace:             namespace.GetName(),
		NamespaceId:           namespace.GetId(),
		OrchestratorComponent: false,
		Replicas:              2,
		Labels:                map[string]string{"k8s-app": "sherlock-holmes"},
		PodLabels:             map[string]string{"k8s-app": "sherlock-holmes"},
		LabelSelector:         &storage.LabelSelector{MatchLabels: map[string]string{"k8s-app": "sherlock-holmes"}},
		Created:               &types.Timestamp{Seconds: 1643589436},
		ClusterId:             namespace.GetClusterId(),
		ClusterName:           namespace.GetClusterName(),
		Containers: []*storage.Container{
			{
				Id: "2edd1e07-2b5a-4f04-8582-42db7fbc9ce7",
				Config: &storage.ContainerConfig{
					Args: []string{"--investigate-dubious-story"},
				},
				Image: &storage.ContainerImage{
					Id:             GetImageSherlockHolmes1().GetId(),
					Name:           GetImageSherlockHolmes1().GetName(),
					NotPullable:    false,
					IsClusterLocal: false,
				},
				SecurityContext: &storage.SecurityContext{
					Privileged:               false,
					Selinux:                  nil,
					DropCapabilities:         []string{"all"},
					AddCapabilities:          []string{"strong_talent_for_observation"},
					ReadOnlyRootFilesystem:   true,
					SeccompProfile:           nil,
					AllowPrivilegeEscalation: false,
				},
				Volumes:        nil,
				Ports:          nil,
				Secrets:        nil,
				Resources:      nil,
				Name:           "sherlockholmes",
				LivenessProbe:  &storage.LivenessProbe{Defined: true},
				ReadinessProbe: &storage.ReadinessProbe{Defined: true},
			},
		},
		Annotations:                   nil,
		Priority:                      3,
		Inactive:                      false,
		ImagePullSecrets:              nil,
		ServiceAccount:                "",
		ServiceAccountPermissionLevel: storage.PermissionLevel_DEFAULT,
		AutomountServiceAccountToken:  true,
		HostNetwork:                   false,
		HostPid:                       false,
		HostIpc:                       false,
		RuntimeClass:                  "",
		Tolerations:                   nil,
		Ports:                         nil,
		StateTimestamp:                1654762976894737,
		RiskScore:                     1.9846836,
	}
}

// GetDeploymentDoctorJekyll2 provides a pseudo-realistic deployment for connected datastore integration testing.
func GetDeploymentDoctorJekyll2(id string, namespace *storage.NamespaceMetadata) *storage.Deployment {
	return &storage.Deployment{
		Id:                    id,
		Name:                  "doctor-jekyll-deployment",
		Hash:                  0,
		Type:                  "Deployment",
		Namespace:             namespace.GetName(),
		NamespaceId:           namespace.GetId(),
		OrchestratorComponent: false,
		Replicas:              2,
		Labels:                map[string]string{"k8s-app": "mr-hyde"},
		PodLabels:             map[string]string{"k8s-app": "mr-hyde"},
		LabelSelector:         &storage.LabelSelector{MatchLabels: map[string]string{"k8s-app": "mr-hyde"}},
		Created:               &types.Timestamp{Seconds: 1643589436},
		ClusterId:             namespace.GetClusterId(),
		ClusterName:           namespace.GetClusterName(),
		Containers: []*storage.Container{
			{
				Id: "2edd1e07-2b5a-4f04-8582-42db7fbc9ce7",
				Config: &storage.ContainerConfig{
					Args: []string{"--tries-to-find-refined-special-crystals"},
				},
				Image: &storage.ContainerImage{
					Id:             GetImageDoctorJekyll2().GetId(),
					Name:           GetImageDoctorJekyll2().GetName(),
					NotPullable:    false,
					IsClusterLocal: false,
				},
				SecurityContext: &storage.SecurityContext{
					Privileged:               false,
					Selinux:                  nil,
					DropCapabilities:         []string{"all"},
					AddCapabilities:          []string{"strong_talent_for_observation"},
					ReadOnlyRootFilesystem:   true,
					SeccompProfile:           nil,
					AllowPrivilegeEscalation: false,
				},
				Volumes:        nil,
				Ports:          nil,
				Secrets:        nil,
				Resources:      nil,
				Name:           "doctorjekyll",
				LivenessProbe:  &storage.LivenessProbe{Defined: true},
				ReadinessProbe: &storage.ReadinessProbe{Defined: true},
			},
		},
		Annotations:                   nil,
		Priority:                      3,
		Inactive:                      false,
		ImagePullSecrets:              nil,
		ServiceAccount:                "",
		ServiceAccountPermissionLevel: storage.PermissionLevel_DEFAULT,
		AutomountServiceAccountToken:  true,
		HostNetwork:                   false,
		HostPid:                       false,
		HostIpc:                       false,
		RuntimeClass:                  "",
		Tolerations:                   nil,
		Ports:                         nil,
		StateTimestamp:                1654762976894737,
		RiskScore:                     1.9846836,
	}
}

// namespace for deployment can be fetched using the namespace fixture GetNamespace(clusterID, clusterName, namespace)

////////////////////
// Node with CVEs //

// Data relationships
//
//       Cluster
//          ^ 1
//          |
//          v *
//         Node   <---------------------------+ *
//          ^ *                               |
//          |                                 |
//  NodeComponentEdge                         |
//          |                                 |
//          v *                               |
//    NodeComponent                           |
// (note: in rocksdb+bleve connected datastore,      NodeCVEEdge
// this is actually ImageComponent)       (removed)
//          ^ *                               |
//          |                                 |
// NodeComponentCVEEdge                       |
// (note: in rocksdb+bleve connected datastore,           |
// this is actually ComponentCVEEdge)         |
//          |                                 |
//          v *                               |
//         CVE   <----------------------------+ *
//
// For testing purposes, a graph of objects like the one below could be used.
//
// Cluster1 -- Node1 --+--> NodeComponent1 --+--> NodeCVE1
//                     |                     |
//                     |                     +-------+
//                     |                             |
//                     |                             v
//                     |                          NodeCVE2
//                     |                             ^
//                     |                             |
//                     |                             +--------+
//                     |                                      |
//                     +--> NodeComponent2 -----> NodeCVE3    |
//                     |                                      |
//                     +----------+                           |
//                                |          +--> NodeCVE4    |
//                                v          |                |
//                          NodeComponent3 --+                |
//                                ^          |                |
//                                |          +--> NodeCVE5    |
//                     +----------+                           |
//                     |                                      |
//                     +--> NodeComponent4                    |
//                     |                                      |
//                     |                     +----------------+
//                     |                     |
// Cluster2 -- Node2 --+--> NodeComponent5 --+--> NodeCVE6
//                                           |
//                                           +--> NodeCVE7

// GetEmbeddedNodeCVE1234x0001 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE1234x0001() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE1234x0001()
	vulnerability.VulnerabilityType = storage.EmbeddedVulnerability_NODE_VULNERABILITY
	vulnerability.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	}
	return vulnerability
}

// GetEmbeddedNodeCVE4567x0002 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE4567x0002() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE4567x0002()
	vulnerability.VulnerabilityType = storage.EmbeddedVulnerability_NODE_VULNERABILITY
	vulnerability.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	}
	return vulnerability
}

// GetEmbeddedNodeCVE1234x0003 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE1234x0003() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE1234x0003()
	vulnerability.VulnerabilityType = storage.EmbeddedVulnerability_NODE_VULNERABILITY
	vulnerability.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	}
	return vulnerability
}

// GetEmbeddedNodeCVE3456x0004 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE3456x0004() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE3456x0004()
	vulnerability.VulnerabilityType = storage.EmbeddedVulnerability_NODE_VULNERABILITY
	vulnerability.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	}
	return vulnerability
}

// GetEmbeddedNodeCVE3456x0005 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE3456x0005() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE3456x0005()
	vulnerability.VulnerabilityType = storage.EmbeddedVulnerability_NODE_VULNERABILITY
	vulnerability.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	}
	return vulnerability
}

// GetEmbeddedNodeCVE2345x0006 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE2345x0006() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE2345x0006()
	vulnerability.VulnerabilityType = storage.EmbeddedVulnerability_NODE_VULNERABILITY
	vulnerability.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	}
	return vulnerability
}

// GetEmbeddedNodeCVE2345x0007 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE2345x0007() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE2345x0007()
	vulnerability.VulnerabilityType = storage.EmbeddedVulnerability_NODE_VULNERABILITY
	vulnerability.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	}
	return vulnerability
}

// GetEmbeddedNodeComponent1x1 provides a pseudo-realistic node component for connected datastore integration testing.
func GetEmbeddedNodeComponent1x1() *storage.EmbeddedNodeScanComponent {
	return &storage.EmbeddedNodeScanComponent{
		Name:    "scarlet",
		Version: "1.1",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedNodeCVE1234x0001(),
			GetEmbeddedNodeCVE4567x0002(),
		},
		Vulnerabilities: nil,
		Priority:        0,
		SetTopCvss:      &storage.EmbeddedNodeScanComponent_TopCvss{TopCvss: 7.5},
		RiskScore:       0,
	}
}

// GetEmbeddedNodeComponent1x2 provides a pseudo-realistic node component for connected datastore integration testing.
func GetEmbeddedNodeComponent1x2() *storage.EmbeddedNodeScanComponent {
	return &storage.EmbeddedNodeScanComponent{
		Name:    "baskerville",
		Version: "1.2",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedNodeCVE1234x0003(),
		},
		Vulnerabilities: nil,
		Priority:        0,
		SetTopCvss:      &storage.EmbeddedNodeScanComponent_TopCvss{TopCvss: 7.5},
		RiskScore:       0,
	}
}

// GetEmbeddedNodeComponent1s2x3 provides a pseudo-realistic node component for connected datastore integration testing.
func GetEmbeddedNodeComponent1s2x3() *storage.EmbeddedNodeScanComponent {
	return &storage.EmbeddedNodeScanComponent{
		Name:    "downtown-london",
		Version: "1s.2-3",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedNodeCVE3456x0004(),
			GetEmbeddedNodeCVE3456x0005(),
		},
		Vulnerabilities: nil,
		Priority:        0,
		SetTopCvss:      &storage.EmbeddedNodeScanComponent_TopCvss{TopCvss: 7.5},
		RiskScore:       0,
	}
}

// GetEmbeddedNodeComponent2x4 provides a pseudo-realistic node component for connected datastore integration testing.
func GetEmbeddedNodeComponent2x4() *storage.EmbeddedNodeScanComponent {
	return &storage.EmbeddedNodeScanComponent{
		Name:            "dr-jekyll-medecine-practice",
		Version:         "2.4",
		Vulns:           []*storage.EmbeddedVulnerability{},
		Vulnerabilities: nil,
		Priority:        0,
		SetTopCvss:      &storage.EmbeddedNodeScanComponent_TopCvss{TopCvss: 0.0},
		RiskScore:       0,
	}
}

// GetEmbeddedNodeComponent2x5 provides a pseudo-realistic node component for connected datastore integration testing.
func GetEmbeddedNodeComponent2x5() *storage.EmbeddedNodeScanComponent {
	return &storage.EmbeddedNodeScanComponent{
		Name:    "mr-hyde-secret-entrance",
		Version: "2.5",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedNodeCVE4567x0002(),
			GetEmbeddedNodeCVE2345x0006(),
			GetEmbeddedNodeCVE2345x0007(),
		},
		Vulnerabilities: nil,
		Priority:        0,
		SetTopCvss:      &storage.EmbeddedNodeScanComponent_TopCvss{TopCvss: 7.8},
		RiskScore:       0,
	}
}

// GetScopedNode1 provides a pseudo-realistic node with scoping information matching the input for
// connected datastore integration testing.
func GetScopedNode1(nodeID string, clusterID string) *storage.Node {
	node := &storage.Node{
		Id:                      nodeID,
		Name:                    "sherlock-holmes",
		Taints:                  nil,
		ClusterId:               clusterID,
		ClusterName:             "test-cluster",
		Labels:                  nil,
		Annotations:             nil,
		JoinedAt:                &types.Timestamp{Seconds: 1643789433},
		InternalIpAddresses:     nil,
		ExternalIpAddresses:     nil,
		ContainerRuntimeVersion: "",
		ContainerRuntime: &storage.ContainerRuntimeInfo{
			Type:    storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
			Version: "20.10.10",
		},
		KernelVersion:    "",
		OsImage:          "",
		KubeletVersion:   "",
		KubeProxyVersion: "",
		LastUpdated:      nil,
		K8SUpdated:       nil,
		Scan: &storage.NodeScan{
			ScanTime:        &types.Timestamp{Seconds: 1654154292, Nanos: 870002400},
			OperatingSystem: "Linux",
			Components: []*storage.EmbeddedNodeScanComponent{
				GetEmbeddedNodeComponent1x1(),
				GetEmbeddedNodeComponent1x2(),
				GetEmbeddedNodeComponent1s2x3(),
			},
			Notes: nil,
		},
		SetComponents: &storage.Node_Components{Components: 3},
		SetCves:       &storage.Node_Cves{Cves: 5},
		SetFixable:    &storage.Node_FixableCves{FixableCves: 2},
		Priority:      0,
		RiskScore:     1.275,
		SetTopCvss:    &storage.Node_TopCvss{TopCvss: 7.5},
		Notes:         nil,
	}
	converter.FillV2NodeVulnerabilities(node)
	return node
}

// GetScopedNode2 provides a pseudo-realistic node with scoping information matching the input for
// connected datastore integration testing.
func GetScopedNode2(nodeID string, clusterID string) *storage.Node {
	node := &storage.Node{
		Id:                      nodeID,
		Name:                    "dr-jekyll",
		Taints:                  nil,
		ClusterId:               clusterID,
		ClusterName:             "test-cluster",
		Labels:                  nil,
		Annotations:             nil,
		JoinedAt:                &types.Timestamp{Seconds: 1643789433},
		InternalIpAddresses:     nil,
		ExternalIpAddresses:     nil,
		ContainerRuntimeVersion: "",
		ContainerRuntime: &storage.ContainerRuntimeInfo{
			Type:    storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
			Version: "20.10.10",
		},
		KernelVersion:    "",
		OperatingSystem:  "Docker Desktop",
		OsImage:          "",
		KubeletVersion:   "",
		KubeProxyVersion: "",
		LastUpdated:      nil,
		K8SUpdated:       nil,
		Scan: &storage.NodeScan{
			ScanTime:        &types.Timestamp{Seconds: 1654154292, Nanos: 870002400},
			OperatingSystem: "Linux",
			Components: []*storage.EmbeddedNodeScanComponent{
				GetEmbeddedNodeComponent1s2x3(),
				GetEmbeddedNodeComponent2x4(),
				GetEmbeddedNodeComponent2x5(),
			},
			Notes: nil,
		},
		SetComponents: &storage.Node_Components{Components: 3},
		SetCves:       &storage.Node_Cves{Cves: 5},
		SetFixable:    &storage.Node_FixableCves{FixableCves: 2},
		Priority:      0,
		RiskScore:     2.375,
		SetTopCvss:    &storage.Node_TopCvss{TopCvss: 7.8},
		Notes:         nil,
	}
	converter.FillV2NodeVulnerabilities(node)
	return node
}

////////////////////////
// Clusters with CVEs //

// Data relationships
//
//       Cluster
//          ^ *
//          |
//    ClusterCVEEdge
//          |
//          v *
//         CVE
//
// For testing purposes, a graph of objects like the one below could be used.
//
// Cluster1 --+--> ClusterCVE1
//            |
//            |
//            +----------+
//                       |
//                       v
//                   ClusterCVE2
//                       ^
//                       |
//            +----------+
//            |
// Cluster2 --+--> ClusterCVE3
//

// GetEmbeddedClusterCVE1234x0001 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedClusterCVE1234x0001() *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:          "CVE-1234-0001",
		Cvss:         5.8,
		Summary:      "Find some inspiring quote on an evil topic to insert here.",
		Link:         "book://author/title",
		SetFixedBy:   &storage.EmbeddedVulnerability_FixedBy{FixedBy: ""},
		ScoreVersion: storage.EmbeddedVulnerability_V2,
		CvssV2: &storage.CVSSV2{
			Vector:              "AV:N/AC:M/Au:N/C:P/I:P/A:N",
			AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
			AccessComplexity:    storage.CVSSV2_ACCESS_MEDIUM,
			Authentication:      storage.CVSSV2_AUTH_NONE,
			Confidentiality:     storage.CVSSV2_IMPACT_PARTIAL,
			Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
			Availability:        storage.CVSSV2_IMPACT_NONE,
			ExploitabilityScore: 8.6,
			ImpactScore:         4.9,
			Score:               5.8,
			Severity:            storage.CVSSV2_MEDIUM,
		},
		CvssV3:            nil,
		PublishedOn:       &types.Timestamp{Seconds: 1234567890},
		LastModified:      &types.Timestamp{Seconds: 1235467890},
		VulnerabilityType: storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{
			storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY,
		},
		Suppressed:            false,
		SuppressActivation:    nil,
		SuppressExpiry:        nil,
		FirstSystemOccurrence: &types.Timestamp{Seconds: 1243567890},
		FirstImageOccurrence:  &types.Timestamp{Seconds: 1245367890},
		Severity:              storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		State:                 storage.VulnerabilityState_OBSERVED,
	}
}

// GetEmbeddedClusterCVE4567x0002 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedClusterCVE4567x0002() *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:          "CVE-4567-0002",
		Cvss:         7.5,
		Summary:      "Find some inspiring quote on an evil topic to insert here.",
		Link:         "book://author/title",
		SetFixedBy:   &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.1.1"},
		ScoreVersion: storage.EmbeddedVulnerability_V3,
		CvssV2: &storage.CVSSV2{
			Vector:              "AV:N/AC:L/Au:N/C:N/I:P/A:N",
			AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
			AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
			Authentication:      storage.CVSSV2_AUTH_NONE,
			Confidentiality:     storage.CVSSV2_IMPACT_NONE,
			Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
			Availability:        storage.CVSSV2_IMPACT_NONE,
			ExploitabilityScore: 10.0,
			ImpactScore:         2.9,
			Score:               5.0,
			Severity:            storage.CVSSV2_MEDIUM,
		},
		CvssV3: &storage.CVSSV3{
			Vector:              "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
			ExploitabilityScore: 3.9,
			ImpactScore:         3.6,
			AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
			AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
			PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
			UserInteraction:     storage.CVSSV3_UI_NONE,
			Scope:               storage.CVSSV3_UNCHANGED,
			Confidentiality:     storage.CVSSV3_IMPACT_NONE,
			Integrity:           storage.CVSSV3_IMPACT_HIGH,
			Availability:        storage.CVSSV3_IMPACT_NONE,
			Score:               7.5,
			Severity:            storage.CVSSV3_HIGH,
		},
		PublishedOn:       &types.Timestamp{Seconds: 1234567890},
		LastModified:      &types.Timestamp{Seconds: 1235467890},
		VulnerabilityType: storage.EmbeddedVulnerability_ISTIO_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{
			storage.EmbeddedVulnerability_K8S_VULNERABILITY,
			storage.EmbeddedVulnerability_ISTIO_VULNERABILITY,
			storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY,
		},
		Suppressed:            false,
		SuppressActivation:    nil,
		SuppressExpiry:        nil,
		FirstSystemOccurrence: &types.Timestamp{Seconds: 1243567890},
		FirstImageOccurrence:  &types.Timestamp{Seconds: 1245367890},
		Severity:              storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		State:                 storage.VulnerabilityState_OBSERVED,
	}
}

// GetEmbeddedClusterCVE2345x0003 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedClusterCVE2345x0003() *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:          "CVE-2345-0003",
		Cvss:         7.8,
		Summary:      "Find some inspiring quote on an evil topic to insert here.",
		Link:         "book://author/title",
		SetFixedBy:   &storage.EmbeddedVulnerability_FixedBy{FixedBy: ""},
		ScoreVersion: storage.EmbeddedVulnerability_V3,
		CvssV2: &storage.CVSSV2{
			Vector:              "AV:N/AC:M/Au:N/C:P/I:P/A:P",
			AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
			AccessComplexity:    storage.CVSSV2_ACCESS_MEDIUM,
			Authentication:      storage.CVSSV2_AUTH_NONE,
			Confidentiality:     storage.CVSSV2_IMPACT_PARTIAL,
			Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
			Availability:        storage.CVSSV2_IMPACT_PARTIAL,
			ExploitabilityScore: 8.6,
			ImpactScore:         6.4,
			Score:               6.8,
			Severity:            storage.CVSSV2_MEDIUM,
		},
		CvssV3: &storage.CVSSV3{
			Vector:              "CVSS:3.0/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
			ExploitabilityScore: 1.8,
			ImpactScore:         5.9,
			AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
			AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
			PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
			UserInteraction:     storage.CVSSV3_UI_REQUIRED,
			Scope:               storage.CVSSV3_UNCHANGED,
			Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
			Integrity:           storage.CVSSV3_IMPACT_HIGH,
			Availability:        storage.CVSSV3_IMPACT_HIGH,
			Score:               7.8,
			Severity:            storage.CVSSV3_HIGH,
		},
		PublishedOn:       &types.Timestamp{Seconds: 1234567890},
		LastModified:      &types.Timestamp{Seconds: 1235467890},
		VulnerabilityType: storage.EmbeddedVulnerability_K8S_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{
			storage.EmbeddedVulnerability_K8S_VULNERABILITY,
		},
		Suppressed:            false,
		SuppressActivation:    nil,
		SuppressExpiry:        nil,
		FirstSystemOccurrence: &types.Timestamp{Seconds: 1243567890},
		FirstImageOccurrence:  &types.Timestamp{Seconds: 1245367890},
		Severity:              storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		State:                 storage.VulnerabilityState_OBSERVED,
	}
}
