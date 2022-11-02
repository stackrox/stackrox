package convert

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/scanner/cpe/nvdtoolscache"
	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/component"
	"github.com/stackrox/scanner/pkg/nvd"
	"github.com/stackrox/scanner/pkg/types"
)

var (
	errSourceTypesMismatch = errors.New("Number of source types in proto and Go are not equal")

	// SourceTypeToProtoMap converts a component.SourceType to a v1.SourceType.
	SourceTypeToProtoMap = func() map[component.SourceType]v1.SourceType {
		numComponentSourceTypes := int(component.SentinelEndSourceType) - int(component.UnsetSourceType)
		if numComponentSourceTypes != len(v1.SourceType_value) {
			utils.CrashOnError(errSourceTypesMismatch)
		}

		m := make(map[component.SourceType]v1.SourceType, numComponentSourceTypes)
		for name, val := range v1.SourceType_value {
			normalizedName := strings.ToLower(strings.TrimSuffix(name, "_SOURCE_TYPE"))
			for sourceType := component.UnsetSourceType; sourceType < component.SentinelEndSourceType; sourceType++ {
				if strings.HasPrefix(strings.ToLower(sourceType.String()), normalizedName) {
					m[sourceType] = v1.SourceType(val)
				}
			}
		}
		if len(m) != numComponentSourceTypes {
			utils.CrashOnError(errSourceTypesMismatch)
		}
		return m
	}()
)

// Metadata converts from types.Metadata to v1.Metadata
func Metadata(m *types.Metadata) *v1.Metadata {
	if m.IsNilOrEmpty() {
		return nil
	}
	metadata := &v1.Metadata{
		PublishedDateTime:    m.PublishedDateTime,
		LastModifiedDateTime: m.LastModifiedDateTime,
	}
	if m.CVSSv2.Vectors != "" {
		cvssV2 := m.CVSSv2
		metadata.CvssV2 = &v1.CVSSMetadata{
			Vector:              cvssV2.Vectors,
			Score:               float32(cvssV2.Score),
			ExploitabilityScore: float32(cvssV2.ExploitabilityScore),
			ImpactScore:         float32(cvssV2.ImpactScore),
		}
	}
	if m.CVSSv3.Vectors != "" {
		cvssV3 := m.CVSSv3
		metadata.CvssV3 = &v1.CVSSMetadata{
			Vector:              cvssV3.Vectors,
			Score:               float32(cvssV3.Score),
			ExploitabilityScore: float32(cvssV3.ExploitabilityScore),
			ImpactScore:         float32(cvssV3.ImpactScore),
		}
	}

	return metadata
}

// MetadataMap converts the internal map[string]interface{} into the API metadata
func MetadataMap(metadataMap map[string]interface{}) (*v1.Metadata, error) {
	var metadataBytes interface{}
	if metadata, exists := metadataMap["Red Hat"]; exists {
		metadataBytes = metadata
	} else if metadata, exists := metadataMap["NVD"]; exists {
		metadataBytes = metadata
	}

	d, err := json.Marshal(&metadataBytes)
	if err != nil {
		return nil, err
	}

	var m types.Metadata
	if err := json.Unmarshal(d, &m); err != nil {
		return nil, err
	}
	return Metadata(&m), err
}

// NVDVulns converts the NVD vuln structure into the API Vulnerability
func NVDVulns(nvdVulns []*nvdtoolscache.NVDCVEItemWithFixedIn) ([]*v1.Vulnerability, error) {
	vulns := make([]*v1.Vulnerability, 0, len(nvdVulns))
	for _, vuln := range nvdVulns {
		m := types.ConvertNVDMetadata(vuln.NVDCVEFeedJSON10DefCVEItem)
		if m.IsNilOrEmpty() {
			log.Warnf("Metadata empty or nil for %v; skipping...", vuln.CVE.CVEDataMeta.ID)
			continue
		}
		vulns = append(vulns, &v1.Vulnerability{
			Name:        vuln.CVE.CVEDataMeta.ID,
			Description: types.ConvertNVDSummary(vuln.NVDCVEFeedJSON10DefCVEItem),
			Link:        nvd.Link(vuln.CVE.CVEDataMeta.ID),
			MetadataV2:  Metadata(m),
			FixedBy:     vuln.FixedIn,
			Severity:    string(DatabaseSeverityToSeverity(m.GetDatabaseSeverity())),
		})
	}

	return vulns, nil
}

// LanguageComponents converts components into gRPC language components.
func LanguageComponents(components []*component.Component) []*v1.LanguageComponent {
	languageComponents := make([]*v1.LanguageComponent, 0, len(components))
	for _, c := range components {
		languageComponent := &v1.LanguageComponent{
			Type:     SourceTypeToProtoMap[c.SourceType],
			Name:     c.Name,
			Version:  c.Version,
			Location: c.Location,
			AddedBy:  c.AddedBy,
		}

		switch c.SourceType {
		case component.JavaSourceType:
			javaMetadata := c.JavaPkgMetadata
			if javaMetadata == nil {
				log.Warnf("Java package %s:%s at %s is invalid; skipping...", c.Name, c.Version, c.Location)
				continue
			} else {
				languageComponent.Language = &v1.LanguageComponent_Java{
					Java: &v1.JavaComponent{
						ImplementationVersion: javaMetadata.ImplementationVersion,
						MavenVersion:          javaMetadata.MavenVersion,
						Origins:               javaMetadata.Origins,
						SpecificationVersion:  javaMetadata.SpecificationVersion,
						BundleName:            javaMetadata.BundleName,
					},
				}
			}
		case component.PythonSourceType:
			pythonMetadata := c.PythonPkgMetadata
			if pythonMetadata == nil {
				log.Warnf("Python package %s:%s at %s is invalid; skipping...", c.Name, c.Version, c.Location)
				continue
			} else {
				languageComponent.Language = &v1.LanguageComponent_Python{
					Python: &v1.PythonComponent{
						Homepage:    pythonMetadata.Homepage,
						AuthorEmail: pythonMetadata.AuthorEmail,
						DownloadUrl: pythonMetadata.DownloadURL,
						Summary:     pythonMetadata.Summary,
						Description: pythonMetadata.Description,
					},
				}
			}
		}

		languageComponents = append(languageComponents, languageComponent)
	}

	return languageComponents
}
