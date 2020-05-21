package m32tom33

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// earliestCVEScanTimes holds the earliest scan time that a CVE was seen. We use this to fill in the CreatedAt times for
// CVEs when they are pulled out of an image.
var earliestCVEScanTimes = make(map[string]*types.Timestamp)

// Split splits the input image into a set of parts.
func Split(image *storage.Image) ImageParts {
	parts := ImageParts{
		image: image.Clone(),
	}

	// These need to be called in order.
	parts.listImage = splitListImage(parts)
	parts.children = splitComponents(parts)

	// Clear components in the top level image.
	if parts.image.GetScan() != nil {
		parts.image.Scan.Components = nil
	}
	return parts
}

func splitListImage(parts ImageParts) *storage.ListImage {
	return convertImageToListImage(parts.image)
}

func splitComponents(parts ImageParts) []ComponentParts {
	ret := make([]ComponentParts, 0, len(parts.image.GetScan().GetComponents()))
	for _, component := range parts.image.GetScan().GetComponents() {
		var cp ComponentParts
		cp.component = generateImageComponent(component)
		cp.edge = generateImageComponentEdge(parts.image, cp.component, component)
		cp.children = splitCVEs(parts, cp, component)

		ret = append(ret, cp)
	}
	return ret
}

func splitCVEs(parts ImageParts, component ComponentParts, embedded *storage.EmbeddedImageScanComponent) []CVEParts {
	ret := make([]CVEParts, 0, len(embedded.GetVulns()))
	for _, cve := range embedded.GetVulns() {
		var cp CVEParts
		cp.cve = generateCVE(parts.image, cve)
		cp.edge = generateComponentCVEEdge(component.component, cp.cve, cve)

		ret = append(ret, cp)
	}
	return ret
}

func generateComponentCVEEdge(convertedComponent *storage.ImageComponent, convertedCVE *storage.CVE, embedded *storage.EmbeddedVulnerability) *storage.ComponentCVEEdge {
	ret := &storage.ComponentCVEEdge{
		Id:        encodeIDPair(convertedComponent.GetId(), convertedCVE.GetId()),
		IsFixable: embedded.GetFixedBy() != "",
	}
	if ret.IsFixable {
		ret.HasFixedBy = &storage.ComponentCVEEdge_FixedBy{
			FixedBy: embedded.GetFixedBy(),
		}
	}
	return ret
}

func generateImageComponent(from *storage.EmbeddedImageScanComponent) *storage.ImageComponent {
	ret := &storage.ImageComponent{
		Id:        encodeIDPair(from.GetName(), from.GetVersion()),
		Name:      from.GetName(),
		Version:   from.GetVersion(),
		License:   from.GetLicense().Clone(),
		RiskScore: from.GetRiskScore(),
	}

	if from.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.ImageComponent_TopCvss{TopCvss: from.GetTopCvss()}
	}
	return ret
}

func generateImageComponentEdge(image *storage.Image, converted *storage.ImageComponent, embedded *storage.EmbeddedImageScanComponent) *storage.ImageComponentEdge {
	ret := &storage.ImageComponentEdge{
		Id: encodeIDPair(image.GetId(), converted.GetId()),
	}
	if embedded.HasLayerIndex != nil {
		ret.HasLayerIndex = &storage.ImageComponentEdge_LayerIndex{
			LayerIndex: embedded.GetLayerIndex(),
		}
	}
	return ret
}

func generateCVE(img *storage.Image, from *storage.EmbeddedVulnerability) *storage.CVE {
	var earliesScan = earliestCVEScanTimes[from.GetCve()]
	if earliesScan == nil || earliesScan.Compare(img.GetScan().GetScanTime()) > 0 {
		earliesScan = img.GetScan().GetScanTime()
		earliestCVEScanTimes[from.GetCve()] = earliesScan
	}
	ret := &storage.CVE{
		Type:         storage.CVE_IMAGE_CVE,
		Id:           from.GetCve(),
		Cvss:         from.GetCvss(),
		Summary:      from.GetSummary(),
		Link:         from.GetLink(),
		PublishedOn:  from.GetPublishedOn(),
		LastModified: from.GetLastModified(),
		CreatedAt:    earliesScan,
		CvssV2:       from.GetCvssV2(),
		CvssV3:       from.GetCvssV3(),
	}
	if ret.CvssV3 != nil {
		ret.ScoreVersion = storage.CVE_V3
		ret.ImpactScore = from.GetCvssV3().GetImpactScore()
	} else if ret.CvssV2 != nil {
		ret.ScoreVersion = storage.CVE_V2
		ret.ImpactScore = from.GetCvssV2().GetImpactScore()
	}
	return ret
}

func convertImageToListImage(i *storage.Image) *storage.ListImage {
	listImage := &storage.ListImage{
		Id:          i.GetId(),
		Name:        i.GetName().GetFullName(),
		Created:     i.GetMetadata().GetV1().GetCreated(),
		LastUpdated: i.GetLastUpdated(),
	}
	if i.GetSetComponents() != nil {
		listImage.SetComponents = &storage.ListImage_Components{
			Components: i.GetComponents(),
		}
	}
	if i.GetSetCves() != nil {
		listImage.SetCves = &storage.ListImage_Cves{
			Cves: i.GetCves(),
		}
	}
	if i.GetSetFixable() != nil {
		listImage.SetFixable = &storage.ListImage_FixableCves{
			FixableCves: i.GetFixableCves(),
		}
	}
	return listImage
}
