package clair

import (
	"encoding/json"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cvss/cvssv2"
	"github.com/stackrox/rox/pkg/cvss/cvssv3"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/scans"
	clairV1 "github.com/stackrox/scanner/api/v1"
	clairConvert "github.com/stackrox/scanner/api/v1/convert"
	clientMetadata "github.com/stackrox/scanner/pkg/clairify/client/metadata"
	"github.com/stackrox/scanner/pkg/component"
)

var (
	log = logging.LoggerForModule()

	// VersionFormatsToSource maps common package formats to their respective source types.
	VersionFormatsToSource = map[string]storage.SourceType{
		component.GemSourceType.String():               storage.SourceType_RUBY,
		component.JavaSourceType.String():              storage.SourceType_JAVA,
		component.NPMSourceType.String():               storage.SourceType_NODEJS,
		component.PythonSourceType.String():            storage.SourceType_PYTHON,
		component.DotNetCoreRuntimeSourceType.String(): storage.SourceType_DOTNETCORERUNTIME,
	}
)

type metadata struct {
	PublishedOn  string `json:"PublishedDateTime"`
	LastModified string `json:"LastModifiedDateTime"`
	CvssV2       *cvss  `json:"CVSSv2"`
	CvssV3       *cvss  `json:"CVSSv3"`
}

type cvss struct {
	Score               float32
	Vectors             string
	ExploitabilityScore float32
	ImpactScore         float32
}

// SeverityToStorageSeverity converts the given string representation of a severity into a storage.VulnerabilitySeverity.
func SeverityToStorageSeverity(severity string) storage.VulnerabilitySeverity {
	switch severity {
	case string(clairConvert.UnknownSeverity):
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	case string(clairConvert.LowSeverity):
		return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	case string(clairConvert.ModerateSeverity):
		return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case string(clairConvert.ImportantSeverity):
		return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case string(clairConvert.CriticalSeverity):
		return storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	}
	return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
}

// ConvertVulnerability converts a clair vulnerability to a proto vulnerability
func ConvertVulnerability(v clairV1.Vulnerability) *storage.EmbeddedVulnerability {
	var vulnMetadataMap interface{}
	var link string
	if metadata, ok := v.Metadata[clientMetadata.NVD]; ok {
		vulnMetadataMap = metadata
		link = scans.GetVulnLink(v.Name)
	} else if metadata, ok := v.Metadata[clientMetadata.RedHat]; ok {
		vulnMetadataMap = metadata
		link = scans.GetRedHatVulnLink(v.Name)
	} else {
		return nil
	}

	if v.Link == "" {
		v.Link = link
	}
	vul := &storage.EmbeddedVulnerability{}
	vul.SetCve(v.Name)
	vul.SetSummary(v.Description)
	vul.SetLink(v.Link)
	vul.Set_FixedBy(v.FixedBy)
	vul.SetVulnerabilityType(storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	vul.SetSeverity(SeverityToStorageSeverity(v.Severity))

	d, err := json.Marshal(vulnMetadataMap)
	if err != nil {
		return vul
	}
	var m metadata
	if err := json.Unmarshal(d, &m); err != nil {
		return vul
	}
	vul.SetPublishedOn(protoconv.ConvertTimeString(m.PublishedOn))
	vul.SetLastModified(protoconv.ConvertTimeString(m.LastModified))

	if m.CvssV2 != nil && m.CvssV2.Vectors != "" {
		if cvssV2, err := cvssv2.ParseCVSSV2(m.CvssV2.Vectors); err == nil {
			cvssV2.SetExploitabilityScore(m.CvssV2.ExploitabilityScore)
			cvssV2.SetImpactScore(m.CvssV2.ImpactScore)
			cvssV2.SetScore(m.CvssV2.Score)

			vul.SetCvssV2(cvssV2)
			// This sets the top level score for use in policies. It will be overwritten if v3 exists
			vul.SetCvss(m.CvssV2.Score)
			vul.SetScoreVersion(storage.EmbeddedVulnerability_V2)
			vul.GetCvssV2().SetSeverity(cvssv2.Severity(vul.GetCvss()))
		} else {
			log.Error(err)
		}
	}

	if m.CvssV3 != nil && m.CvssV3.Vectors != "" {
		if cvssV3, err := cvssv3.ParseCVSSV3(m.CvssV3.Vectors); err == nil {
			cvssV3.SetExploitabilityScore(m.CvssV3.ExploitabilityScore)
			cvssV3.SetImpactScore(m.CvssV3.ImpactScore)
			cvssV3.SetScore(m.CvssV3.Score)

			vul.SetCvssV3(cvssV3)
			vul.SetCvss(m.CvssV3.Score)
			vul.SetScoreVersion(storage.EmbeddedVulnerability_V3)
			vul.GetCvssV3().SetSeverity(cvssv3.Severity(vul.GetCvss()))
		} else {
			log.Error(err)
		}
	}
	return vul
}

func convertFeature(feature clairV1.Feature, os string) *storage.EmbeddedImageScanComponent {
	component := &storage.EmbeddedImageScanComponent{}
	component.SetName(feature.Name)
	component.SetVersion(feature.Version)
	component.SetLocation(feature.Location)
	component.SetFixedBy(feature.FixedBy)

	if source, ok := VersionFormatsToSource[feature.VersionFormat]; ok {
		component.SetSource(source)
	}
	component.SetVulns(make([]*storage.EmbeddedVulnerability, 0, len(feature.Vulnerabilities)))
	for _, v := range feature.Vulnerabilities {
		if convertedVuln := ConvertVulnerability(v); convertedVuln != nil {
			component.SetVulns(append(component.GetVulns(), convertedVuln))
		}
	}
	// TODO:  Figure out what is happening with Active Vuln Management
	if features.ActiveVulnMgmt.Enabled() && !features.FlattenCVEData.Enabled() {
		executables := make([]*storage.EmbeddedImageScanComponent_Executable, 0, len(feature.Executables))
		for _, executable := range feature.Executables {
			imageComponentIds := make([]string, 0, len(executable.GetRequiredFeatures()))
			for _, f := range executable.GetRequiredFeatures() {
				imageComponentIds = append(imageComponentIds, scancomponent.ComponentID(f.GetName(), f.GetVersion(), os))
			}
			exec := &storage.EmbeddedImageScanComponent_Executable{}
			exec.SetPath(executable.GetPath())
			exec.SetDependencies(imageComponentIds)
			executables = append(executables, exec)
		}
		component.SetExecutables(executables)
	}

	return component
}

// BuildSHAToIndexMap takes image metadata and maps each layer's SHA to its index.
func BuildSHAToIndexMap(metadata *storage.ImageMetadata) map[string]int32 {
	layerSHAToIndex := make(map[string]int32)

	if metadata.GetV2() != nil {
		var layerIdx int
		for i, l := range metadata.GetV1().GetLayers() {
			if !l.GetEmpty() {
				if layerIdx >= len(metadata.GetLayerShas()) {
					log.Error("More layers than expected when correlating V2 instructions to V1 layers")
					break
				}
				sha := metadata.GetLayerShas()[layerIdx]
				layerSHAToIndex[sha] = int32(i)
				layerIdx++
			}
		}
	} else {
		// If it's V1 then we should have a 1:1 mapping of layer SHAs to the layerOrdering slice
		for i := range metadata.GetV1().GetLayers() {
			if i >= len(metadata.GetLayerShas()) {
				log.Error("More layers than expected when correlating V1 instructions to V1 layers")
				break
			}
			layerSHAToIndex[metadata.GetLayerShas()[i]] = int32(i)
		}
	}
	return layerSHAToIndex
}

// ConvertFeatures converts clair features to proto components
func ConvertFeatures(image *storage.Image, features []clairV1.Feature, os string) (components []*storage.EmbeddedImageScanComponent) {
	layerSHAToIndex := BuildSHAToIndexMap(image.GetMetadata())

	components = make([]*storage.EmbeddedImageScanComponent, 0, len(features))
	for _, feature := range features {
		convertedComponent := convertFeature(feature, os)
		if val, ok := layerSHAToIndex[feature.AddedBy]; ok {
			convertedComponent.SetLayerIndex(val)
		}
		components = append(components, convertedComponent)
	}
	return
}
