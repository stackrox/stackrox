package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/nodes/converter"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/proto"
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
	cVSSV2 := &storage.CVSSV2{}
	cVSSV2.SetVector("AV:N/AC:M/Au:N/C:P/I:P/A:N")
	cVSSV2.SetAttackVector(storage.CVSSV2_ATTACK_NETWORK)
	cVSSV2.SetAccessComplexity(storage.CVSSV2_ACCESS_MEDIUM)
	cVSSV2.SetAuthentication(storage.CVSSV2_AUTH_NONE)
	cVSSV2.SetConfidentiality(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetIntegrity(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetAvailability(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetExploitabilityScore(8.6)
	cVSSV2.SetImpactScore(4.9)
	cVSSV2.SetScore(5.8)
	cVSSV2.SetSeverity(storage.CVSSV2_MEDIUM)
	ev := &storage.EmbeddedVulnerability{}
	ev.SetCve("CVE-1234-0001")
	ev.SetCvss(5.8)
	ev.SetSummary("Find some inspiring quote on an evil topic to insert here.")
	ev.SetLink("book://author/title")
	ev.Set_FixedBy("")
	ev.SetScoreVersion(storage.EmbeddedVulnerability_V2)
	ev.SetCvssV2(cVSSV2)
	ev.ClearCvssV3()
	ev.SetPublishedOn(protocompat.GetProtoTimestampFromSeconds(1234567890))
	ev.SetLastModified(protocompat.GetProtoTimestampFromSeconds(1235467890))
	ev.SetVulnerabilityType(storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	ev.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
	})
	ev.SetSuppressed(false)
	ev.ClearSuppressActivation()
	ev.ClearSuppressExpiry()
	ev.SetFirstSystemOccurrence(protocompat.GetProtoTimestampFromSeconds(1243567890))
	ev.SetFirstImageOccurrence(protocompat.GetProtoTimestampFromSeconds(1245367890))
	ev.SetSeverity(storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY)
	ev.SetState(storage.VulnerabilityState_OBSERVED)
	return ev
}

// GetEmbeddedImageCVE4567x0002 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE4567x0002() *storage.EmbeddedVulnerability {
	cVSSV2 := &storage.CVSSV2{}
	cVSSV2.SetVector("AV:N/AC:L/Au:N/C:N/I:P/A:N")
	cVSSV2.SetAttackVector(storage.CVSSV2_ATTACK_NETWORK)
	cVSSV2.SetAccessComplexity(storage.CVSSV2_ACCESS_LOW)
	cVSSV2.SetAuthentication(storage.CVSSV2_AUTH_NONE)
	cVSSV2.SetConfidentiality(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetIntegrity(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetAvailability(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetExploitabilityScore(10.0)
	cVSSV2.SetImpactScore(2.9)
	cVSSV2.SetScore(5.0)
	cVSSV2.SetSeverity(storage.CVSSV2_MEDIUM)
	cVSSV3 := &storage.CVSSV3{}
	cVSSV3.SetVector("CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N")
	cVSSV3.SetExploitabilityScore(3.9)
	cVSSV3.SetImpactScore(3.6)
	cVSSV3.SetAttackVector(storage.CVSSV3_ATTACK_NETWORK)
	cVSSV3.SetAttackComplexity(storage.CVSSV3_COMPLEXITY_LOW)
	cVSSV3.SetPrivilegesRequired(storage.CVSSV3_PRIVILEGE_NONE)
	cVSSV3.SetUserInteraction(storage.CVSSV3_UI_NONE)
	cVSSV3.SetScope(storage.CVSSV3_UNCHANGED)
	cVSSV3.SetConfidentiality(storage.CVSSV3_IMPACT_NONE)
	cVSSV3.SetIntegrity(storage.CVSSV3_IMPACT_HIGH)
	cVSSV3.SetAvailability(storage.CVSSV3_IMPACT_NONE)
	cVSSV3.SetScore(7.5)
	cVSSV3.SetSeverity(storage.CVSSV3_HIGH)
	ev := &storage.EmbeddedVulnerability{}
	ev.SetCve("CVE-4567-0002")
	ev.SetCvss(7.5)
	ev.SetSummary("Find some inspiring quote on an evil topic to insert here.")
	ev.SetLink("book://author/title")
	ev.Set_FixedBy("1.1.1")
	ev.SetScoreVersion(storage.EmbeddedVulnerability_V3)
	ev.SetCvssV2(cVSSV2)
	ev.SetCvssV3(cVSSV3)
	ev.SetPublishedOn(protocompat.GetProtoTimestampFromSeconds(1234567890))
	ev.SetLastModified(protocompat.GetProtoTimestampFromSeconds(1235467890))
	ev.SetVulnerabilityType(storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	ev.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
	})
	ev.SetSuppressed(false)
	ev.ClearSuppressActivation()
	ev.ClearSuppressExpiry()
	ev.SetFirstSystemOccurrence(protocompat.GetProtoTimestampFromSeconds(1243567890))
	ev.SetFirstImageOccurrence(protocompat.GetProtoTimestampFromSeconds(1245367890))
	ev.SetSeverity(storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY)
	ev.SetState(storage.VulnerabilityState_OBSERVED)
	return ev
}

// GetEmbeddedImageCVE1234x0003 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE1234x0003() *storage.EmbeddedVulnerability {
	cVSSV2 := &storage.CVSSV2{}
	cVSSV2.SetVector("AV:N/AC:L/Au:N/C:N/I:N/A:P")
	cVSSV2.SetAttackVector(storage.CVSSV2_ATTACK_NETWORK)
	cVSSV2.SetAccessComplexity(storage.CVSSV2_ACCESS_LOW)
	cVSSV2.SetAuthentication(storage.CVSSV2_AUTH_NONE)
	cVSSV2.SetConfidentiality(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetIntegrity(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetAvailability(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetExploitabilityScore(10.0)
	cVSSV2.SetImpactScore(2.9)
	cVSSV2.SetScore(5.0)
	cVSSV2.SetSeverity(storage.CVSSV2_MEDIUM)
	cVSSV3 := &storage.CVSSV3{}
	cVSSV3.SetVector("CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H")
	cVSSV3.SetExploitabilityScore(3.9)
	cVSSV3.SetImpactScore(3.6)
	cVSSV3.SetAttackVector(storage.CVSSV3_ATTACK_NETWORK)
	cVSSV3.SetAttackComplexity(storage.CVSSV3_COMPLEXITY_LOW)
	cVSSV3.SetPrivilegesRequired(storage.CVSSV3_PRIVILEGE_NONE)
	cVSSV3.SetUserInteraction(storage.CVSSV3_UI_NONE)
	cVSSV3.SetScope(storage.CVSSV3_UNCHANGED)
	cVSSV3.SetConfidentiality(storage.CVSSV3_IMPACT_NONE)
	cVSSV3.SetIntegrity(storage.CVSSV3_IMPACT_NONE)
	cVSSV3.SetAvailability(storage.CVSSV3_IMPACT_HIGH)
	cVSSV3.SetScore(7.5)
	cVSSV3.SetSeverity(storage.CVSSV3_HIGH)
	ev := &storage.EmbeddedVulnerability{}
	ev.SetCve("CVE-1234-0003")
	ev.SetCvss(7.5)
	ev.SetSummary("Find some inspiring quote on an evil topic to insert here.")
	ev.SetLink("book://author/title")
	ev.Set_FixedBy("")
	ev.SetScoreVersion(storage.EmbeddedVulnerability_V3)
	ev.SetCvssV2(cVSSV2)
	ev.SetCvssV3(cVSSV3)
	ev.SetPublishedOn(protocompat.GetProtoTimestampFromSeconds(1234567890))
	ev.SetLastModified(protocompat.GetProtoTimestampFromSeconds(1235467890))
	ev.SetVulnerabilityType(storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	ev.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
	})
	ev.SetSuppressed(false)
	ev.ClearSuppressActivation()
	ev.ClearSuppressExpiry()
	ev.SetFirstSystemOccurrence(protocompat.GetProtoTimestampFromSeconds(1243567890))
	ev.SetFirstImageOccurrence(protocompat.GetProtoTimestampFromSeconds(1245367890))
	ev.SetSeverity(storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY)
	ev.SetState(storage.VulnerabilityState_OBSERVED)
	return ev
}

// GetEmbeddedImageCVE3456x0004 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE3456x0004() *storage.EmbeddedVulnerability {
	cVSSV2 := &storage.CVSSV2{}
	cVSSV2.SetVector("AV:N/AC:M/Au:N/C:N/I:N/A:P")
	cVSSV2.SetAttackVector(storage.CVSSV2_ATTACK_NETWORK)
	cVSSV2.SetAccessComplexity(storage.CVSSV2_ACCESS_MEDIUM)
	cVSSV2.SetAuthentication(storage.CVSSV2_AUTH_NONE)
	cVSSV2.SetConfidentiality(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetIntegrity(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetAvailability(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetExploitabilityScore(8.6)
	cVSSV2.SetImpactScore(2.9)
	cVSSV2.SetScore(4.3)
	cVSSV2.SetSeverity(storage.CVSSV2_MEDIUM)
	cVSSV3 := &storage.CVSSV3{}
	cVSSV3.SetVector("CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H")
	cVSSV3.SetExploitabilityScore(2.2)
	cVSSV3.SetImpactScore(3.6)
	cVSSV3.SetAttackVector(storage.CVSSV3_ATTACK_NETWORK)
	cVSSV3.SetAttackComplexity(storage.CVSSV3_COMPLEXITY_HIGH)
	cVSSV3.SetPrivilegesRequired(storage.CVSSV3_PRIVILEGE_NONE)
	cVSSV3.SetUserInteraction(storage.CVSSV3_UI_NONE)
	cVSSV3.SetScope(storage.CVSSV3_UNCHANGED)
	cVSSV3.SetConfidentiality(storage.CVSSV3_IMPACT_NONE)
	cVSSV3.SetIntegrity(storage.CVSSV3_IMPACT_NONE)
	cVSSV3.SetAvailability(storage.CVSSV3_IMPACT_HIGH)
	cVSSV3.SetScore(5.9)
	cVSSV3.SetSeverity(storage.CVSSV3_MEDIUM)
	ev := &storage.EmbeddedVulnerability{}
	ev.SetCve("CVE-3456-0004")
	ev.SetCvss(7.5)
	ev.SetSummary("Find some inspiring quote on an evil topic to insert here.")
	ev.SetLink("book://author/title")
	ev.Set_FixedBy("")
	ev.SetScoreVersion(storage.EmbeddedVulnerability_V3)
	ev.SetCvssV2(cVSSV2)
	ev.SetCvssV3(cVSSV3)
	ev.SetPublishedOn(protocompat.GetProtoTimestampFromSeconds(1234567890))
	ev.SetLastModified(protocompat.GetProtoTimestampFromSeconds(1235467890))
	ev.SetVulnerabilityType(storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	ev.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
	})
	ev.SetSuppressed(false)
	ev.ClearSuppressActivation()
	ev.ClearSuppressExpiry()
	ev.SetFirstSystemOccurrence(protocompat.GetProtoTimestampFromSeconds(1243567890))
	ev.SetFirstImageOccurrence(protocompat.GetProtoTimestampFromSeconds(1245367890))
	ev.SetSeverity(storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY)
	ev.SetState(storage.VulnerabilityState_OBSERVED)
	return ev
}

// GetEmbeddedImageCVE3456x0005 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE3456x0005() *storage.EmbeddedVulnerability {
	cVSSV2 := &storage.CVSSV2{}
	cVSSV2.SetVector("AV:L/AC:L/Au:N/C:P/I:P/A:P")
	cVSSV2.SetAttackVector(storage.CVSSV2_ATTACK_LOCAL)
	cVSSV2.SetAccessComplexity(storage.CVSSV2_ACCESS_LOW)
	cVSSV2.SetAuthentication(storage.CVSSV2_AUTH_NONE)
	cVSSV2.SetConfidentiality(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetIntegrity(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetAvailability(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetExploitabilityScore(3.9)
	cVSSV2.SetImpactScore(6.4)
	cVSSV2.SetScore(4.6)
	cVSSV2.SetSeverity(storage.CVSSV2_MEDIUM)
	cVSSV3 := &storage.CVSSV3{}
	cVSSV3.SetVector("CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:L/A:L")
	cVSSV3.SetExploitabilityScore(1.8)
	cVSSV3.SetImpactScore(3.4)
	cVSSV3.SetAttackVector(storage.CVSSV3_ATTACK_LOCAL)
	cVSSV3.SetAttackComplexity(storage.CVSSV3_COMPLEXITY_LOW)
	cVSSV3.SetPrivilegesRequired(storage.CVSSV3_PRIVILEGE_LOW)
	cVSSV3.SetUserInteraction(storage.CVSSV3_UI_NONE)
	cVSSV3.SetScope(storage.CVSSV3_UNCHANGED)
	cVSSV3.SetConfidentiality(storage.CVSSV3_IMPACT_LOW)
	cVSSV3.SetIntegrity(storage.CVSSV3_IMPACT_LOW)
	cVSSV3.SetAvailability(storage.CVSSV3_IMPACT_LOW)
	cVSSV3.SetScore(5.3)
	cVSSV3.SetSeverity(storage.CVSSV3_MEDIUM)
	ev := &storage.EmbeddedVulnerability{}
	ev.SetCve("CVE-3456-0005")
	ev.SetCvss(5.3)
	ev.SetSummary("Find some inspiring quote on an evil topic to insert here.")
	ev.SetLink("book://author/title")
	ev.Set_FixedBy("")
	ev.SetScoreVersion(storage.EmbeddedVulnerability_V3)
	ev.SetCvssV2(cVSSV2)
	ev.SetCvssV3(cVSSV3)
	ev.SetPublishedOn(protocompat.GetProtoTimestampFromSeconds(1234567890))
	ev.SetLastModified(protocompat.GetProtoTimestampFromSeconds(1235467890))
	ev.SetVulnerabilityType(storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	ev.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
	})
	ev.SetSuppressed(false)
	ev.ClearSuppressActivation()
	ev.ClearSuppressExpiry()
	ev.SetFirstSystemOccurrence(protocompat.GetProtoTimestampFromSeconds(1243567890))
	ev.SetFirstImageOccurrence(protocompat.GetProtoTimestampFromSeconds(1245367890))
	ev.SetSeverity(storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY)
	ev.SetState(storage.VulnerabilityState_OBSERVED)
	return ev
}

// GetEmbeddedImageCVE2345x0006 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE2345x0006() *storage.EmbeddedVulnerability {
	cVSSV2 := &storage.CVSSV2{}
	cVSSV2.SetVector("AV:N/AC:M/Au:N/C:P/I:P/A:P")
	cVSSV2.SetAttackVector(storage.CVSSV2_ATTACK_NETWORK)
	cVSSV2.SetAccessComplexity(storage.CVSSV2_ACCESS_MEDIUM)
	cVSSV2.SetAuthentication(storage.CVSSV2_AUTH_NONE)
	cVSSV2.SetConfidentiality(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetIntegrity(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetAvailability(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetExploitabilityScore(8.6)
	cVSSV2.SetImpactScore(6.4)
	cVSSV2.SetScore(6.8)
	cVSSV2.SetSeverity(storage.CVSSV2_MEDIUM)
	cVSSV3 := &storage.CVSSV3{}
	cVSSV3.SetVector("CVSS:3.0/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H")
	cVSSV3.SetExploitabilityScore(1.8)
	cVSSV3.SetImpactScore(5.9)
	cVSSV3.SetAttackVector(storage.CVSSV3_ATTACK_LOCAL)
	cVSSV3.SetAttackComplexity(storage.CVSSV3_COMPLEXITY_LOW)
	cVSSV3.SetPrivilegesRequired(storage.CVSSV3_PRIVILEGE_NONE)
	cVSSV3.SetUserInteraction(storage.CVSSV3_UI_REQUIRED)
	cVSSV3.SetScope(storage.CVSSV3_UNCHANGED)
	cVSSV3.SetConfidentiality(storage.CVSSV3_IMPACT_HIGH)
	cVSSV3.SetIntegrity(storage.CVSSV3_IMPACT_HIGH)
	cVSSV3.SetAvailability(storage.CVSSV3_IMPACT_HIGH)
	cVSSV3.SetScore(7.8)
	cVSSV3.SetSeverity(storage.CVSSV3_HIGH)
	ev := &storage.EmbeddedVulnerability{}
	ev.SetCve("CVE-2345-0006")
	ev.SetCvss(7.8)
	ev.SetSummary("Find some inspiring quote on an evil topic to insert here.")
	ev.SetLink("book://author/title")
	ev.Set_FixedBy("")
	ev.SetScoreVersion(storage.EmbeddedVulnerability_V3)
	ev.SetCvssV2(cVSSV2)
	ev.SetCvssV3(cVSSV3)
	ev.SetPublishedOn(protocompat.GetProtoTimestampFromSeconds(1234567890))
	ev.SetLastModified(protocompat.GetProtoTimestampFromSeconds(1235467890))
	ev.SetVulnerabilityType(storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	ev.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
	})
	ev.SetSuppressed(false)
	ev.ClearSuppressActivation()
	ev.ClearSuppressExpiry()
	ev.SetFirstSystemOccurrence(protocompat.GetProtoTimestampFromSeconds(1243567890))
	ev.SetFirstImageOccurrence(protocompat.GetProtoTimestampFromSeconds(1245367890))
	ev.SetSeverity(storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY)
	ev.SetState(storage.VulnerabilityState_OBSERVED)
	return ev
}

// GetEmbeddedImageCVE2345x0007 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedImageCVE2345x0007() *storage.EmbeddedVulnerability {
	cVSSV2 := &storage.CVSSV2{}
	cVSSV2.SetVector("AV:N/AC:M/Au:N/C:N/I:P/A:N")
	cVSSV2.SetAttackVector(storage.CVSSV2_ATTACK_NETWORK)
	cVSSV2.SetAccessComplexity(storage.CVSSV2_ACCESS_MEDIUM)
	cVSSV2.SetAuthentication(storage.CVSSV2_AUTH_NONE)
	cVSSV2.SetConfidentiality(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetIntegrity(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetAvailability(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetExploitabilityScore(8.6)
	cVSSV2.SetImpactScore(2.9)
	cVSSV2.SetScore(4.3)
	cVSSV2.SetSeverity(storage.CVSSV2_MEDIUM)
	cVSSV3 := &storage.CVSSV3{}
	cVSSV3.SetVector("CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N")
	cVSSV3.SetExploitabilityScore(2.2)
	cVSSV3.SetImpactScore(3.6)
	cVSSV3.SetAttackVector(storage.CVSSV3_ATTACK_NETWORK)
	cVSSV3.SetAttackComplexity(storage.CVSSV3_COMPLEXITY_HIGH)
	cVSSV3.SetPrivilegesRequired(storage.CVSSV3_PRIVILEGE_NONE)
	cVSSV3.SetUserInteraction(storage.CVSSV3_UI_NONE)
	cVSSV3.SetScope(storage.CVSSV3_UNCHANGED)
	cVSSV3.SetConfidentiality(storage.CVSSV3_IMPACT_NONE)
	cVSSV3.SetIntegrity(storage.CVSSV3_IMPACT_HIGH)
	cVSSV3.SetAvailability(storage.CVSSV3_IMPACT_NONE)
	cVSSV3.SetScore(5.9)
	cVSSV3.SetSeverity(storage.CVSSV3_MEDIUM)
	ev := &storage.EmbeddedVulnerability{}
	ev.SetCve("CVE-2345-0007")
	ev.SetCvss(5.9)
	ev.SetSummary("Find some inspiring quote on an evil topic to insert here.")
	ev.SetLink("book://author/title")
	ev.Set_FixedBy("2.5.6")
	ev.SetScoreVersion(storage.EmbeddedVulnerability_V3)
	ev.SetCvssV2(cVSSV2)
	ev.SetCvssV3(cVSSV3)
	ev.SetPublishedOn(protocompat.GetProtoTimestampFromSeconds(1234567890))
	ev.SetLastModified(protocompat.GetProtoTimestampFromSeconds(1235467890))
	ev.SetVulnerabilityType(storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	ev.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
	})
	ev.SetSuppressed(false)
	ev.ClearSuppressActivation()
	ev.ClearSuppressExpiry()
	ev.SetFirstSystemOccurrence(protocompat.GetProtoTimestampFromSeconds(1243567890))
	ev.SetFirstImageOccurrence(protocompat.GetProtoTimestampFromSeconds(1245367890))
	ev.SetSeverity(storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY)
	ev.SetState(storage.VulnerabilityState_OBSERVED)
	return ev
}

// GetEmbeddedImageComponent1x1 provides a pseudo-realistic image component for connected datastore integration testing.
func GetEmbeddedImageComponent1x1() *storage.EmbeddedImageScanComponent {
	eisc := &storage.EmbeddedImageScanComponent{}
	eisc.SetName("scarlet")
	eisc.SetVersion("1.1")
	eisc.ClearLicense()
	eisc.SetVulns([]*storage.EmbeddedVulnerability{
		GetEmbeddedImageCVE1234x0001(),
		GetEmbeddedImageCVE4567x0002(),
	})
	eisc.SetLayerIndex(0)
	eisc.SetPriority(0)
	eisc.SetSource(storage.SourceType_OS)
	eisc.SetLocation("")
	eisc.Set_TopCvss(7.5)
	eisc.SetRiskScore(2.154)
	eisc.SetFixedBy("1.1.1")
	eisc.SetExecutables([]*storage.EmbeddedImageScanComponent_Executable{})
	return eisc
}

// GetEmbeddedImageComponent1x2 provides a pseudo-realistic image component for connected datastore integration testing.
func GetEmbeddedImageComponent1x2() *storage.EmbeddedImageScanComponent {
	ee := &storage.EmbeddedImageScanComponent_Executable{}
	ee.SetPath("/horrific/hound")
	eisc := &storage.EmbeddedImageScanComponent{}
	eisc.SetName("baskerville")
	eisc.SetVersion("1.2")
	eisc.ClearLicense()
	eisc.SetVulns([]*storage.EmbeddedVulnerability{
		GetEmbeddedImageCVE1234x0003(),
	})
	eisc.SetLayerIndex(1)
	eisc.SetPriority(0)
	eisc.SetSource(storage.SourceType_PYTHON)
	eisc.SetLocation("")
	eisc.Set_TopCvss(7.5)
	eisc.SetRiskScore(1.1625)
	eisc.SetFixedBy("1.2.5")
	eisc.SetExecutables([]*storage.EmbeddedImageScanComponent_Executable{
		ee,
	})
	return eisc
}

// GetEmbeddedImageComponent1s2x3 provides a pseudo-realistic image component for connected datastore integration testing.
func GetEmbeddedImageComponent1s2x3() *storage.EmbeddedImageScanComponent {
	eisc := &storage.EmbeddedImageScanComponent{}
	eisc.SetName("downtown-london")
	eisc.SetVersion("1s.2-3")
	eisc.ClearLicense()
	eisc.SetVulns([]*storage.EmbeddedVulnerability{
		GetEmbeddedImageCVE3456x0004(),
		GetEmbeddedImageCVE3456x0005(),
	})
	eisc.SetLayerIndex(1)
	eisc.SetPriority(0)
	eisc.SetSource(storage.SourceType_JAVA)
	eisc.SetLocation("")
	eisc.Set_TopCvss(7.5)
	eisc.SetRiskScore(1.1625)
	eisc.SetFixedBy("")
	eisc.SetExecutables([]*storage.EmbeddedImageScanComponent_Executable{})
	return eisc
}

// GetEmbeddedImageComponent2x4 provides a pseudo-realistic image component for connected datastore integration testing.
func GetEmbeddedImageComponent2x4() *storage.EmbeddedImageScanComponent {
	eisc := &storage.EmbeddedImageScanComponent{}
	eisc.SetName("dr-jekyll-medecine-practice")
	eisc.SetVersion("2.4")
	eisc.ClearLicense()
	eisc.SetVulns([]*storage.EmbeddedVulnerability{})
	eisc.SetLayerIndex(0)
	eisc.SetPriority(0)
	eisc.SetSource(storage.SourceType_INFRASTRUCTURE)
	eisc.SetLocation("")
	eisc.Set_TopCvss(0.0)
	eisc.SetRiskScore(0.0)
	eisc.SetFixedBy("")
	eisc.SetExecutables([]*storage.EmbeddedImageScanComponent_Executable{})
	return eisc
}

// GetEmbeddedImageComponent2x5 provides a pseudo-realistic image component for connected datastore integration testing.
func GetEmbeddedImageComponent2x5() *storage.EmbeddedImageScanComponent {
	return storage.EmbeddedImageScanComponent_builder{
		Name:    "mr-hyde-secret-entrance",
		Version: "2.5",
		License: nil,
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE4567x0002(),
			GetEmbeddedImageCVE2345x0006(),
			GetEmbeddedImageCVE2345x0007(),
		},
		LayerIndex: proto.Int32(2),
		Priority:   0,
		Source:     storage.SourceType_RUBY,
		Location:   "",
		TopCvss:    proto.Float32(7.8),
		RiskScore:  0.0,
		FixedBy:    "2.5.6",
		Executables: []*storage.EmbeddedImageScanComponent_Executable{
			storage.EmbeddedImageScanComponent_Executable_builder{
				Path: "/murderous/cane",
				Dependencies: []string{
					"experimental-powder",
				},
			}.Build(),
		},
	}.Build()
}

// GetImageSherlockHolmes1 provides a pseudo-realistic image for connected datastore integration testing.
func GetImageSherlockHolmes1() *storage.Image {
	return storage.Image_builder{
		Id: "sha256:50fa59cca653c51d194974830826ff7a9d9095175f78caf40d5423d3fb12c4f7",
		Name: storage.ImageName_builder{
			Registry: "baker.st",
			Remote:   "sherlock/holmes",
			Tag:      "v1",
			FullName: "baker.st/sherlock/holmes:v1",
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Digest:  "sha256:0a488a3872bfcd9e79a3575b5c273b01c01a21b16e86213a26eb7f3ab540eb84",
				Created: protocompat.GetProtoTimestampFromSecondsAndNanos(1553642092, 227945051),
				Author:  "Sir Arthur Conan Doyle",
				Layers: []*storage.ImageLayer{
					storage.ImageLayer_builder{
						Instruction: "COPY",
						Value:       "/ / # buildkit",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553640086, 106246179),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: " /usr/local/bin/ # buildkit",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553640126, 263243615),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "ADD",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553640134, 213199897),
					}.Build(),
				},
				User:       "root",
				Command:    nil,
				Entrypoint: nil,
				Volumes:    nil,
				Labels:     nil,
			}.Build(),
			V2: storage.V2Metadata_builder{Digest: "sha256:4d818f38fa9dcbf41e7c255f276a72e5c471c1523b6f755a344bac04652351dd"}.Build(),
			LayerShas: []string{
				"sha256:50fa59cca653c51d194974830826ff7a9d9095175f78caf40d5423d3fb12c4f7",
				"sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				"sha256:8e2ee98ae01ebe81fe221f5a444cab18c8f7a26cd00ce1ef23cf7432feef99b4",
			},
			DataSource: storage.DataSource_builder{
				Id:   "13e92196-8216-4714-9ac7-fac779bb973b",
				Name: "Sir Arthur Conan Doyle",
			}.Build(),
			Version: 0,
		}.Build(),
		Scan: storage.ImageScan_builder{
			ScannerVersion: "2.24.0-11-g05cf175999",
			ScanTime:       protocompat.GetProtoTimestampFromSecondsAndNanos(1654154310, 970783800),
			Components: []*storage.EmbeddedImageScanComponent{
				GetEmbeddedImageComponent1x1(),
				GetEmbeddedImageComponent1x2(),
				GetEmbeddedImageComponent1s2x3(),
			},
			OperatingSystem: "crime-stories",
			DataSource: storage.DataSource_builder{
				Id:   "169b0d3f-8277-4900-bbce-1127077defae",
				Name: "Stackrox Scanner",
			}.Build(),
			Notes: []storage.ImageScan_Note{},
		}.Build(),
		SignatureVerificationData: nil,
		Signature:                 nil,
		Components:                proto.Int32(3),
		Cves:                      proto.Int32(5),
		FixableCves:               proto.Int32(2),
		LastUpdated:               protocompat.GetProtoTimestampFromSecondsAndNanos(1654154313, 67882700),
		NotPullable:               false,
		IsClusterLocal:            false,
		Priority:                  0,
		RiskScore:                 1.5,
		TopCvss:                   proto.Float32(7.5),
		Notes: []storage.Image_Note{
			storage.Image_MISSING_SIGNATURE_VERIFICATION_DATA,
			storage.Image_MISSING_SIGNATURE,
		},
	}.Build()
}

// GetImageDoctorJekyll2 provides a pseudo-realistic image for connected datastore integration testing.
func GetImageDoctorJekyll2() *storage.Image {
	return storage.Image_builder{
		Id: "sha256:835762dc5388a591ecf31540eaeb14ec8bc96ad48a3bd11fdef77b7106111eec",
		Name: storage.ImageName_builder{
			Registry: "book.worm",
			Remote:   "doctor/jekyll",
			Tag:      "v2",
			FullName: "book.worm/doctor/jekyll:v2",
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Digest:  "sha256:9fe0366ee2eead5a66948f853ebedae5464361b5ffb166980db355d294a971ff",
				Created: protocompat.GetProtoTimestampFromSecondsAndNanos(1553642392, 877872600),
				Author:  "Sir Arthur Conan Doyle",
				Layers: []*storage.ImageLayer{
					storage.ImageLayer_builder{
						Instruction: "COPY",
						Value:       "/ / # buildkit",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553641386, 227945051),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: " /usr/local/bin/ # buildkit",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553641426, 106246179),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "ADD",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553641534, 302497847),
					}.Build(),
				},
				User:       "root",
				Command:    nil,
				Entrypoint: nil,
				Volumes:    nil,
				Labels:     nil,
			}.Build(),
			V2: storage.V2Metadata_builder{Digest: "sha256:1e0ccd4630c681f887d799677a5d846d13e7fb69d4c4e25b899ba12ce804ac06"}.Build(),
			LayerShas: []string{
				"sha256:8d5041d30882e1fff9d4f4f90a72bd22896c8e365ad6095189d70d605bf7d3bd",
				"sha256:9bc15005c6e7e93dcbe4c05f61dae53fcd72ce81aa44b3ae3d989e910ac682b9",
				"sha256:e83e783ca7567afba7ea4541e6233032ca0303b43811539453c36b11c497eda8",
			},
			DataSource: storage.DataSource_builder{
				Id:   "28eacb99-4e61-8be8-c316e6875184",
				Name: "Robert Louis Stevenson",
			}.Build(),
			Version: 0,
		}.Build(),
		Scan: storage.ImageScan_builder{
			ScannerVersion: "2.24.0-11-g05cf175999",
			ScanTime:       protocompat.GetProtoTimestampFromSecondsAndNanos(1654154710, 67882700),
			Components: []*storage.EmbeddedImageScanComponent{
				GetEmbeddedImageComponent1s2x3(),
				GetEmbeddedImageComponent2x4(),
				GetEmbeddedImageComponent2x5(),
			},
			OperatingSystem: "crime-stories",
			DataSource: storage.DataSource_builder{
				Id:   "169b0d3f-8277-4900-bbce-1127077defae",
				Name: "Stackrox Scanner",
			}.Build(),
			Notes: []storage.ImageScan_Note{},
		}.Build(),
		SignatureVerificationData: nil,
		Signature:                 nil,
		Components:                proto.Int32(3),
		Cves:                      proto.Int32(5),
		FixableCves:               proto.Int32(2),
		LastUpdated:               protocompat.GetProtoTimestampFromSecondsAndNanos(1654154413, 970783800),
		NotPullable:               false,
		IsClusterLocal:            false,
		Priority:                  0,
		RiskScore:                 2.375,
		TopCvss:                   proto.Float32(7.8),
		Notes: []storage.Image_Note{
			storage.Image_MISSING_SIGNATURE_VERIFICATION_DATA,
			storage.Image_MISSING_SIGNATURE,
		},
	}.Build()
}

// GetImageV2SherlockHolmes1 provides a pseudo-realistic image (ImageV2) for connected datastore integration testing.
func GetImageV2SherlockHolmes1() *storage.ImageV2 {
	imageName := "baker.st/sherlock/holmes:v1"
	imageSha := "sha256:50fa59cca653c51d194974830826ff7a9d9095175f78caf40d5423d3fb12c4f7"
	return storage.ImageV2_builder{
		Id:     uuid.NewV5FromNonUUIDs(imageName, imageSha).String(),
		Digest: imageSha,
		Name: storage.ImageName_builder{
			Registry: "baker.st",
			Remote:   "sherlock/holmes",
			Tag:      "v1",
			FullName: imageName,
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Digest:  "sha256:0a488a3872bfcd9e79a3575b5c273b01c01a21b16e86213a26eb7f3ab540eb84",
				Created: protocompat.GetProtoTimestampFromSecondsAndNanos(1553642092, 227945051),
				Author:  "Sir Arthur Conan Doyle",
				Layers: []*storage.ImageLayer{
					storage.ImageLayer_builder{
						Instruction: "COPY",
						Value:       "/ / # buildkit",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553640086, 106246179),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: " /usr/local/bin/ # buildkit",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553640126, 263243615),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "ADD",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553640134, 213199897),
					}.Build(),
				},
				User:       "root",
				Command:    nil,
				Entrypoint: nil,
				Volumes:    nil,
				Labels:     nil,
			}.Build(),
			V2: storage.V2Metadata_builder{Digest: "sha256:4d818f38fa9dcbf41e7c255f276a72e5c471c1523b6f755a344bac04652351dd"}.Build(),
			LayerShas: []string{
				"sha256:50fa59cca653c51d194974830826ff7a9d9095175f78caf40d5423d3fb12c4f7",
				"sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				"sha256:8e2ee98ae01ebe81fe221f5a444cab18c8f7a26cd00ce1ef23cf7432feef99b4",
			},
			DataSource: storage.DataSource_builder{
				Id:   "13e92196-8216-4714-9ac7-fac779bb973b",
				Name: "Sir Arthur Conan Doyle",
			}.Build(),
			Version: 0,
		}.Build(),
		Scan: storage.ImageScan_builder{
			ScannerVersion: "2.24.0-11-g05cf175999",
			ScanTime:       protocompat.GetProtoTimestampFromSecondsAndNanos(1654154310, 970783800),
			Components: []*storage.EmbeddedImageScanComponent{
				GetEmbeddedImageComponent1x1(),
				GetEmbeddedImageComponent1x2(),
				GetEmbeddedImageComponent1s2x3(),
			},
			OperatingSystem: "crime-stories",
			DataSource: storage.DataSource_builder{
				Id:   "169b0d3f-8277-4900-bbce-1127077defae",
				Name: "Stackrox Scanner",
			}.Build(),
			Notes: []storage.ImageScan_Note{},
		}.Build(),
		SignatureVerificationData: nil,
		Signature:                 nil,
		ScanStats: storage.ImageV2_ScanStats_builder{
			ComponentCount:  3,
			CveCount:        5,
			FixableCveCount: 1,
		}.Build(),
		LastUpdated:    protocompat.GetProtoTimestampFromSecondsAndNanos(1654154313, 67882700),
		NotPullable:    false,
		IsClusterLocal: false,
		Priority:       0,
		RiskScore:      1.5,
		TopCvss:        7.5,
		Notes: []storage.ImageV2_Note{
			storage.ImageV2_MISSING_SIGNATURE_VERIFICATION_DATA,
			storage.ImageV2_MISSING_SIGNATURE,
		},
	}.Build()
}

// GetImageV2DoctorJekyll2 provides a pseudo-realistic image (ImageV2) for connected datastore integration testing.
func GetImageV2DoctorJekyll2() *storage.ImageV2 {
	imageName := "book.worm/doctor/jekyll:v2"
	imageSha := "sha256:835762dc5388a591ecf31540eaeb14ec8bc96ad48a3bd11fdef77b7106111eec"
	return storage.ImageV2_builder{
		Id: uuid.NewV5FromNonUUIDs(imageName, imageSha).String(),
		Name: storage.ImageName_builder{
			Registry: "book.worm",
			Remote:   "doctor/jekyll",
			Tag:      "v2",
			FullName: imageName,
		}.Build(),
		Digest: imageSha,
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Digest:  "sha256:9fe0366ee2eead5a66948f853ebedae5464361b5ffb166980db355d294a971ff",
				Created: protocompat.GetProtoTimestampFromSecondsAndNanos(1553642392, 877872600),
				Author:  "Sir Arthur Conan Doyle",
				Layers: []*storage.ImageLayer{
					storage.ImageLayer_builder{
						Instruction: "COPY",
						Value:       "/ / # buildkit",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553641386, 227945051),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: " /usr/local/bin/ # buildkit",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553641426, 106246179),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "ADD",
						Value:       "file:4fc310c0cb879c876c5c0f571af765a0d24d36cb9253e0f53a0cda2f7e4c1844 in /",
						Created:     protocompat.GetProtoTimestampFromSecondsAndNanos(1553641534, 302497847),
					}.Build(),
				},
				User:       "root",
				Command:    nil,
				Entrypoint: nil,
				Volumes:    nil,
				Labels:     nil,
			}.Build(),
			V2: storage.V2Metadata_builder{Digest: "sha256:1e0ccd4630c681f887d799677a5d846d13e7fb69d4c4e25b899ba12ce804ac06"}.Build(),
			LayerShas: []string{
				"sha256:8d5041d30882e1fff9d4f4f90a72bd22896c8e365ad6095189d70d605bf7d3bd",
				"sha256:9bc15005c6e7e93dcbe4c05f61dae53fcd72ce81aa44b3ae3d989e910ac682b9",
				"sha256:e83e783ca7567afba7ea4541e6233032ca0303b43811539453c36b11c497eda8",
			},
			DataSource: storage.DataSource_builder{
				Id:   "28eacb99-4e61-8be8-c316e6875184",
				Name: "Robert Louis Stevenson",
			}.Build(),
			Version: 0,
		}.Build(),
		Scan: storage.ImageScan_builder{
			ScannerVersion: "2.24.0-11-g05cf175999",
			ScanTime:       protocompat.GetProtoTimestampFromSecondsAndNanos(1654154710, 67882700),
			Components: []*storage.EmbeddedImageScanComponent{
				GetEmbeddedImageComponent1s2x3(),
				GetEmbeddedImageComponent2x4(),
				GetEmbeddedImageComponent2x5(),
			},
			OperatingSystem: "crime-stories",
			DataSource: storage.DataSource_builder{
				Id:   "169b0d3f-8277-4900-bbce-1127077defae",
				Name: "Stackrox Scanner",
			}.Build(),
			Notes: []storage.ImageScan_Note{},
		}.Build(),
		SignatureVerificationData: nil,
		Signature:                 nil,
		ScanStats: storage.ImageV2_ScanStats_builder{
			ComponentCount:  3,
			CveCount:        5,
			FixableCveCount: 2,
		}.Build(),
		LastUpdated:    protocompat.GetProtoTimestampFromSecondsAndNanos(1654154413, 970783800),
		NotPullable:    false,
		IsClusterLocal: false,
		Priority:       0,
		RiskScore:      2.375,
		TopCvss:        7.8,
		Notes: []storage.ImageV2_Note{
			storage.ImageV2_MISSING_SIGNATURE_VERIFICATION_DATA,
			storage.ImageV2_MISSING_SIGNATURE,
		},
	}.Build()
}

// GetDeploymentSherlockHolmes1 provides a pseudo-realistic deployment for connected datastore integration testing.
func GetDeploymentSherlockHolmes1(id string, namespace *storage.NamespaceMetadata) *storage.Deployment {
	return storage.Deployment_builder{
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
		LabelSelector:         storage.LabelSelector_builder{MatchLabels: map[string]string{"k8s-app": "sherlock-holmes"}}.Build(),
		Created:               protocompat.GetProtoTimestampFromSeconds(1643589436),
		ClusterId:             namespace.GetClusterId(),
		ClusterName:           namespace.GetClusterName(),
		Containers: []*storage.Container{
			storage.Container_builder{
				Id: "2edd1e07-2b5a-4f04-8582-42db7fbc9ce7",
				Config: storage.ContainerConfig_builder{
					Args: []string{"--investigate-dubious-story"},
				}.Build(),
				Image: storage.ContainerImage_builder{
					Id: func() string {
						if !features.FlattenImageData.Enabled() {
							return GetImageSherlockHolmes1().GetId()
						}
						return ""
					}(),
					IdV2: func() string {
						if !features.FlattenImageData.Enabled() {
							return ""
						}
						return GetImageV2SherlockHolmes1().GetId()
					}(),
					Name: func() *storage.ImageName {
						if !features.FlattenImageData.Enabled() {
							return GetImageSherlockHolmes1().GetName()
						}
						return GetImageV2SherlockHolmes1().GetName()
					}(),
					NotPullable:    false,
					IsClusterLocal: false,
				}.Build(),
				SecurityContext: storage.SecurityContext_builder{
					Privileged:               false,
					Selinux:                  nil,
					DropCapabilities:         []string{"all"},
					AddCapabilities:          []string{"strong_talent_for_observation"},
					ReadOnlyRootFilesystem:   true,
					SeccompProfile:           nil,
					AllowPrivilegeEscalation: false,
				}.Build(),
				Volumes:        nil,
				Ports:          nil,
				Secrets:        nil,
				Resources:      nil,
				Name:           "sherlockholmes",
				LivenessProbe:  storage.LivenessProbe_builder{Defined: true}.Build(),
				ReadinessProbe: storage.ReadinessProbe_builder{Defined: true}.Build(),
			}.Build(),
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
	}.Build()
}

// GetDeploymentDoctorJekyll2 provides a pseudo-realistic deployment for connected datastore integration testing.
func GetDeploymentDoctorJekyll2(id string, namespace *storage.NamespaceMetadata) *storage.Deployment {
	return storage.Deployment_builder{
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
		LabelSelector:         storage.LabelSelector_builder{MatchLabels: map[string]string{"k8s-app": "mr-hyde"}}.Build(),
		Created:               protocompat.GetProtoTimestampFromSeconds(1643589436),
		ClusterId:             namespace.GetClusterId(),
		ClusterName:           namespace.GetClusterName(),
		Containers: []*storage.Container{
			storage.Container_builder{
				Id: "2edd1e07-2b5a-4f04-8582-42db7fbc9ce7",
				Config: storage.ContainerConfig_builder{
					Args: []string{"--tries-to-find-refined-special-crystals"},
				}.Build(),
				Image: storage.ContainerImage_builder{
					Id: func() string {
						if !features.FlattenImageData.Enabled() {
							return GetImageDoctorJekyll2().GetId()
						}
						return ""
					}(),
					IdV2: func() string {
						if !features.FlattenImageData.Enabled() {
							return ""
						}
						return GetImageV2DoctorJekyll2().GetId()
					}(),
					Name: func() *storage.ImageName {
						if !features.FlattenImageData.Enabled() {
							return GetImageDoctorJekyll2().GetName()
						}
						return GetImageV2DoctorJekyll2().GetName()
					}(),
					NotPullable:    false,
					IsClusterLocal: false,
				}.Build(),
				SecurityContext: storage.SecurityContext_builder{
					Privileged:               false,
					Selinux:                  nil,
					DropCapabilities:         []string{"all"},
					AddCapabilities:          []string{"strong_talent_for_observation"},
					ReadOnlyRootFilesystem:   true,
					SeccompProfile:           nil,
					AllowPrivilegeEscalation: false,
				}.Build(),
				Volumes:        nil,
				Ports:          nil,
				Secrets:        nil,
				Resources:      nil,
				Name:           "doctorjekyll",
				LivenessProbe:  storage.LivenessProbe_builder{Defined: true}.Build(),
				ReadinessProbe: storage.ReadinessProbe_builder{Defined: true}.Build(),
			}.Build(),
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
	}.Build()
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
	vulnerability.SetVulnerabilityType(storage.EmbeddedVulnerability_NODE_VULNERABILITY)
	vulnerability.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	})
	return vulnerability
}

// GetEmbeddedNodeCVE4567x0002 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE4567x0002() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE4567x0002()
	vulnerability.SetVulnerabilityType(storage.EmbeddedVulnerability_NODE_VULNERABILITY)
	vulnerability.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	})
	return vulnerability
}

// GetEmbeddedNodeCVE1234x0003 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE1234x0003() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE1234x0003()
	vulnerability.SetVulnerabilityType(storage.EmbeddedVulnerability_NODE_VULNERABILITY)
	vulnerability.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	})
	return vulnerability
}

// GetEmbeddedNodeCVE3456x0004 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE3456x0004() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE3456x0004()
	vulnerability.SetVulnerabilityType(storage.EmbeddedVulnerability_NODE_VULNERABILITY)
	vulnerability.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	})
	return vulnerability
}

// GetEmbeddedNodeCVE3456x0005 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE3456x0005() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE3456x0005()
	vulnerability.SetVulnerabilityType(storage.EmbeddedVulnerability_NODE_VULNERABILITY)
	vulnerability.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	})
	return vulnerability
}

// GetEmbeddedNodeCVE2345x0006 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE2345x0006() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE2345x0006()
	vulnerability.SetVulnerabilityType(storage.EmbeddedVulnerability_NODE_VULNERABILITY)
	vulnerability.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	})
	return vulnerability
}

// GetEmbeddedNodeCVE2345x0007 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedNodeCVE2345x0007() *storage.EmbeddedVulnerability {
	vulnerability := GetEmbeddedImageCVE2345x0007()
	vulnerability.SetVulnerabilityType(storage.EmbeddedVulnerability_NODE_VULNERABILITY)
	vulnerability.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	})
	return vulnerability
}

// GetEmbeddedNodeComponent1x1 provides a pseudo-realistic node component for connected datastore integration testing.
func GetEmbeddedNodeComponent1x1() *storage.EmbeddedNodeScanComponent {
	ensc := &storage.EmbeddedNodeScanComponent{}
	ensc.SetName("scarlet")
	ensc.SetVersion("1.1")
	ensc.SetVulns([]*storage.EmbeddedVulnerability{
		GetEmbeddedNodeCVE1234x0001(),
		GetEmbeddedNodeCVE4567x0002(),
	})
	ensc.SetVulnerabilities(nil)
	ensc.SetPriority(0)
	ensc.Set_TopCvss(7.5)
	ensc.SetRiskScore(0)
	return ensc
}

// GetEmbeddedNodeComponent1x2 provides a pseudo-realistic node component for connected datastore integration testing.
func GetEmbeddedNodeComponent1x2() *storage.EmbeddedNodeScanComponent {
	ensc := &storage.EmbeddedNodeScanComponent{}
	ensc.SetName("baskerville")
	ensc.SetVersion("1.2")
	ensc.SetVulns([]*storage.EmbeddedVulnerability{
		GetEmbeddedNodeCVE1234x0003(),
	})
	ensc.SetVulnerabilities(nil)
	ensc.SetPriority(0)
	ensc.Set_TopCvss(7.5)
	ensc.SetRiskScore(0)
	return ensc
}

// GetEmbeddedNodeComponent1s2x3 provides a pseudo-realistic node component for connected datastore integration testing.
func GetEmbeddedNodeComponent1s2x3() *storage.EmbeddedNodeScanComponent {
	ensc := &storage.EmbeddedNodeScanComponent{}
	ensc.SetName("downtown-london")
	ensc.SetVersion("1s.2-3")
	ensc.SetVulns([]*storage.EmbeddedVulnerability{
		GetEmbeddedNodeCVE3456x0004(),
		GetEmbeddedNodeCVE3456x0005(),
	})
	ensc.SetVulnerabilities(nil)
	ensc.SetPriority(0)
	ensc.Set_TopCvss(7.5)
	ensc.SetRiskScore(0)
	return ensc
}

// GetEmbeddedNodeComponent2x4 provides a pseudo-realistic node component for connected datastore integration testing.
func GetEmbeddedNodeComponent2x4() *storage.EmbeddedNodeScanComponent {
	ensc := &storage.EmbeddedNodeScanComponent{}
	ensc.SetName("dr-jekyll-medecine-practice")
	ensc.SetVersion("2.4")
	ensc.SetVulns([]*storage.EmbeddedVulnerability{})
	ensc.SetVulnerabilities(nil)
	ensc.SetPriority(0)
	ensc.Set_TopCvss(0.0)
	ensc.SetRiskScore(0)
	return ensc
}

// GetEmbeddedNodeComponent2x5 provides a pseudo-realistic node component for connected datastore integration testing.
func GetEmbeddedNodeComponent2x5() *storage.EmbeddedNodeScanComponent {
	ensc := &storage.EmbeddedNodeScanComponent{}
	ensc.SetName("mr-hyde-secret-entrance")
	ensc.SetVersion("2.5")
	ensc.SetVulns([]*storage.EmbeddedVulnerability{
		GetEmbeddedNodeCVE4567x0002(),
		GetEmbeddedNodeCVE2345x0006(),
		GetEmbeddedNodeCVE2345x0007(),
	})
	ensc.SetVulnerabilities(nil)
	ensc.SetPriority(0)
	ensc.Set_TopCvss(7.8)
	ensc.SetRiskScore(0)
	return ensc
}

// GetScopedNode1 provides a pseudo-realistic node with scoping information matching the input for
// connected datastore integration testing.
func GetScopedNode1(nodeID string, clusterID string) *storage.Node {
	cri := &storage.ContainerRuntimeInfo{}
	cri.SetType(storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME)
	cri.SetVersion("20.10.10")
	nodeScan := &storage.NodeScan{}
	nodeScan.SetScanTime(protocompat.GetProtoTimestampFromSecondsAndNanos(1654154292, 870002400))
	nodeScan.SetOperatingSystem("Linux")
	nodeScan.SetComponents([]*storage.EmbeddedNodeScanComponent{
		GetEmbeddedNodeComponent1x1(),
		GetEmbeddedNodeComponent1x2(),
		GetEmbeddedNodeComponent1s2x3(),
	})
	nodeScan.SetNotes(nil)
	node := &storage.Node{}
	node.SetId(nodeID)
	node.SetName("sherlock-holmes")
	node.SetTaints(nil)
	node.SetClusterId(clusterID)
	node.SetClusterName("test-cluster")
	node.SetLabels(nil)
	node.SetAnnotations(nil)
	node.SetJoinedAt(protocompat.GetProtoTimestampFromSeconds(1643789433))
	node.SetInternalIpAddresses(nil)
	node.SetExternalIpAddresses(nil)
	node.SetContainerRuntimeVersion("")
	node.SetContainerRuntime(cri)
	node.SetKernelVersion("")
	node.SetOsImage("")
	node.SetKubeletVersion("")
	node.SetKubeProxyVersion("")
	node.ClearLastUpdated()
	node.ClearK8SUpdated()
	node.SetScan(nodeScan)
	node.Set_Components(3)
	node.Set_Cves(5)
	node.SetFixableCves(2)
	node.SetPriority(0)
	node.SetRiskScore(1.275)
	node.Set_TopCvss(7.5)
	node.SetNotes(nil)
	converter.FillV2NodeVulnerabilities(node)
	return node
}

// GetScopedNode2 provides a pseudo-realistic node with scoping information matching the input for
// connected datastore integration testing.
func GetScopedNode2(nodeID string, clusterID string) *storage.Node {
	cri := &storage.ContainerRuntimeInfo{}
	cri.SetType(storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME)
	cri.SetVersion("20.10.10")
	nodeScan := &storage.NodeScan{}
	nodeScan.SetScanTime(protocompat.GetProtoTimestampFromSecondsAndNanos(1654154292, 870002400))
	nodeScan.SetOperatingSystem("Linux")
	nodeScan.SetComponents([]*storage.EmbeddedNodeScanComponent{
		GetEmbeddedNodeComponent1s2x3(),
		GetEmbeddedNodeComponent2x4(),
		GetEmbeddedNodeComponent2x5(),
	})
	nodeScan.SetNotes(nil)
	node := &storage.Node{}
	node.SetId(nodeID)
	node.SetName("dr-jekyll")
	node.SetTaints(nil)
	node.SetClusterId(clusterID)
	node.SetClusterName("test-cluster")
	node.SetLabels(nil)
	node.SetAnnotations(nil)
	node.SetJoinedAt(protocompat.GetProtoTimestampFromSeconds(1643789433))
	node.SetInternalIpAddresses(nil)
	node.SetExternalIpAddresses(nil)
	node.SetContainerRuntimeVersion("")
	node.SetContainerRuntime(cri)
	node.SetKernelVersion("")
	node.SetOperatingSystem("Docker Desktop")
	node.SetOsImage("")
	node.SetKubeletVersion("")
	node.SetKubeProxyVersion("")
	node.ClearLastUpdated()
	node.ClearK8SUpdated()
	node.SetScan(nodeScan)
	node.Set_Components(3)
	node.Set_Cves(5)
	node.SetFixableCves(2)
	node.SetPriority(0)
	node.SetRiskScore(2.375)
	node.Set_TopCvss(7.8)
	node.SetNotes(nil)
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
	cVSSV2 := &storage.CVSSV2{}
	cVSSV2.SetVector("AV:N/AC:M/Au:N/C:P/I:P/A:N")
	cVSSV2.SetAttackVector(storage.CVSSV2_ATTACK_NETWORK)
	cVSSV2.SetAccessComplexity(storage.CVSSV2_ACCESS_MEDIUM)
	cVSSV2.SetAuthentication(storage.CVSSV2_AUTH_NONE)
	cVSSV2.SetConfidentiality(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetIntegrity(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetAvailability(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetExploitabilityScore(8.6)
	cVSSV2.SetImpactScore(4.9)
	cVSSV2.SetScore(5.8)
	cVSSV2.SetSeverity(storage.CVSSV2_MEDIUM)
	ev := &storage.EmbeddedVulnerability{}
	ev.SetCve("CVE-1234-0001")
	ev.SetCvss(5.8)
	ev.SetSummary("Find some inspiring quote on an evil topic to insert here.")
	ev.SetLink("book://author/title")
	ev.Set_FixedBy("")
	ev.SetScoreVersion(storage.EmbeddedVulnerability_V2)
	ev.SetCvssV2(cVSSV2)
	ev.ClearCvssV3()
	ev.SetPublishedOn(protocompat.GetProtoTimestampFromSeconds(1234567890))
	ev.SetLastModified(protocompat.GetProtoTimestampFromSeconds(1235467890))
	ev.SetVulnerabilityType(storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY)
	ev.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY,
	})
	ev.SetSuppressed(false)
	ev.ClearSuppressActivation()
	ev.ClearSuppressExpiry()
	ev.SetFirstSystemOccurrence(protocompat.GetProtoTimestampFromSeconds(1243567890))
	ev.SetFirstImageOccurrence(protocompat.GetProtoTimestampFromSeconds(1245367890))
	ev.SetSeverity(storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY)
	ev.SetState(storage.VulnerabilityState_OBSERVED)
	return ev
}

// GetEmbeddedClusterCVE4567x0002 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedClusterCVE4567x0002() *storage.EmbeddedVulnerability {
	cVSSV2 := &storage.CVSSV2{}
	cVSSV2.SetVector("AV:N/AC:L/Au:N/C:N/I:P/A:N")
	cVSSV2.SetAttackVector(storage.CVSSV2_ATTACK_NETWORK)
	cVSSV2.SetAccessComplexity(storage.CVSSV2_ACCESS_LOW)
	cVSSV2.SetAuthentication(storage.CVSSV2_AUTH_NONE)
	cVSSV2.SetConfidentiality(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetIntegrity(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetAvailability(storage.CVSSV2_IMPACT_NONE)
	cVSSV2.SetExploitabilityScore(10.0)
	cVSSV2.SetImpactScore(2.9)
	cVSSV2.SetScore(5.0)
	cVSSV2.SetSeverity(storage.CVSSV2_MEDIUM)
	cVSSV3 := &storage.CVSSV3{}
	cVSSV3.SetVector("CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N")
	cVSSV3.SetExploitabilityScore(3.9)
	cVSSV3.SetImpactScore(3.6)
	cVSSV3.SetAttackVector(storage.CVSSV3_ATTACK_NETWORK)
	cVSSV3.SetAttackComplexity(storage.CVSSV3_COMPLEXITY_LOW)
	cVSSV3.SetPrivilegesRequired(storage.CVSSV3_PRIVILEGE_NONE)
	cVSSV3.SetUserInteraction(storage.CVSSV3_UI_NONE)
	cVSSV3.SetScope(storage.CVSSV3_UNCHANGED)
	cVSSV3.SetConfidentiality(storage.CVSSV3_IMPACT_NONE)
	cVSSV3.SetIntegrity(storage.CVSSV3_IMPACT_HIGH)
	cVSSV3.SetAvailability(storage.CVSSV3_IMPACT_NONE)
	cVSSV3.SetScore(7.5)
	cVSSV3.SetSeverity(storage.CVSSV3_HIGH)
	ev := &storage.EmbeddedVulnerability{}
	ev.SetCve("CVE-4567-0002")
	ev.SetCvss(7.5)
	ev.SetSummary("Find some inspiring quote on an evil topic to insert here.")
	ev.SetLink("book://author/title")
	ev.Set_FixedBy("1.1.1")
	ev.SetScoreVersion(storage.EmbeddedVulnerability_V3)
	ev.SetCvssV2(cVSSV2)
	ev.SetCvssV3(cVSSV3)
	ev.SetPublishedOn(protocompat.GetProtoTimestampFromSeconds(1234567890))
	ev.SetLastModified(protocompat.GetProtoTimestampFromSeconds(1235467890))
	ev.SetVulnerabilityType(storage.EmbeddedVulnerability_ISTIO_VULNERABILITY)
	ev.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_K8S_VULNERABILITY,
		storage.EmbeddedVulnerability_ISTIO_VULNERABILITY,
		storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY,
	})
	ev.SetSuppressed(false)
	ev.ClearSuppressActivation()
	ev.ClearSuppressExpiry()
	ev.SetFirstSystemOccurrence(protocompat.GetProtoTimestampFromSeconds(1243567890))
	ev.SetFirstImageOccurrence(protocompat.GetProtoTimestampFromSeconds(1245367890))
	ev.SetSeverity(storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY)
	ev.SetState(storage.VulnerabilityState_OBSERVED)
	return ev
}

// GetEmbeddedClusterCVE2345x0003 provides a pseudo-realistic image CVE for connected datastore datastore integration testing.
func GetEmbeddedClusterCVE2345x0003() *storage.EmbeddedVulnerability {
	cVSSV2 := &storage.CVSSV2{}
	cVSSV2.SetVector("AV:N/AC:M/Au:N/C:P/I:P/A:P")
	cVSSV2.SetAttackVector(storage.CVSSV2_ATTACK_NETWORK)
	cVSSV2.SetAccessComplexity(storage.CVSSV2_ACCESS_MEDIUM)
	cVSSV2.SetAuthentication(storage.CVSSV2_AUTH_NONE)
	cVSSV2.SetConfidentiality(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetIntegrity(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetAvailability(storage.CVSSV2_IMPACT_PARTIAL)
	cVSSV2.SetExploitabilityScore(8.6)
	cVSSV2.SetImpactScore(6.4)
	cVSSV2.SetScore(6.8)
	cVSSV2.SetSeverity(storage.CVSSV2_MEDIUM)
	cVSSV3 := &storage.CVSSV3{}
	cVSSV3.SetVector("CVSS:3.0/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H")
	cVSSV3.SetExploitabilityScore(1.8)
	cVSSV3.SetImpactScore(5.9)
	cVSSV3.SetAttackVector(storage.CVSSV3_ATTACK_LOCAL)
	cVSSV3.SetAttackComplexity(storage.CVSSV3_COMPLEXITY_LOW)
	cVSSV3.SetPrivilegesRequired(storage.CVSSV3_PRIVILEGE_NONE)
	cVSSV3.SetUserInteraction(storage.CVSSV3_UI_REQUIRED)
	cVSSV3.SetScope(storage.CVSSV3_UNCHANGED)
	cVSSV3.SetConfidentiality(storage.CVSSV3_IMPACT_HIGH)
	cVSSV3.SetIntegrity(storage.CVSSV3_IMPACT_HIGH)
	cVSSV3.SetAvailability(storage.CVSSV3_IMPACT_HIGH)
	cVSSV3.SetScore(7.8)
	cVSSV3.SetSeverity(storage.CVSSV3_HIGH)
	ev := &storage.EmbeddedVulnerability{}
	ev.SetCve("CVE-2345-0003")
	ev.SetCvss(7.8)
	ev.SetSummary("Find some inspiring quote on an evil topic to insert here.")
	ev.SetLink("book://author/title")
	ev.Set_FixedBy("")
	ev.SetScoreVersion(storage.EmbeddedVulnerability_V3)
	ev.SetCvssV2(cVSSV2)
	ev.SetCvssV3(cVSSV3)
	ev.SetPublishedOn(protocompat.GetProtoTimestampFromSeconds(1234567890))
	ev.SetLastModified(protocompat.GetProtoTimestampFromSeconds(1235467890))
	ev.SetVulnerabilityType(storage.EmbeddedVulnerability_K8S_VULNERABILITY)
	ev.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{
		storage.EmbeddedVulnerability_K8S_VULNERABILITY,
	})
	ev.SetSuppressed(false)
	ev.ClearSuppressActivation()
	ev.ClearSuppressExpiry()
	ev.SetFirstSystemOccurrence(protocompat.GetProtoTimestampFromSeconds(1243567890))
	ev.SetFirstImageOccurrence(protocompat.GetProtoTimestampFromSeconds(1245367890))
	ev.SetSeverity(storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY)
	ev.SetState(storage.VulnerabilityState_OBSERVED)
	return ev
}
