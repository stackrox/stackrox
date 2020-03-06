package m27tom28

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
)

// Split splits the input image into a set of parts.
func Split(image *storage.Image) ImageParts {
	parts := ImageParts{
		image: proto.Clone(image).(*storage.Image),
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
		cp.children = splitCVEs(cp, component)

		ret = append(ret, cp)
	}
	return ret
}

func splitCVEs(component ComponentParts, embedded *storage.EmbeddedImageScanComponent) []CVEParts {
	ret := make([]CVEParts, 0, len(embedded.GetVulns()))
	for _, cve := range embedded.GetVulns() {
		var cp CVEParts
		cp.cve = generateCVE(cve)
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
		License:   proto.Clone(from.GetLicense()).(*storage.License),
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

func generateCVE(from *storage.EmbeddedVulnerability) *storage.CVE {
	ret := &storage.CVE{
		Type:         storage.CVE_IMAGE_CVE,
		Id:           from.GetCve(),
		Cvss:         from.GetCvss(),
		Summary:      from.GetSummary(),
		Link:         from.GetLink(),
		PublishedOn:  from.GetPublishedOn(),
		LastModified: from.GetLastModified(),
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

	if i.GetScan() != nil {
		listImage.SetComponents = &storage.ListImage_Components{
			Components: int32(len(i.GetScan().GetComponents())),
		}
		var numVulns int32
		var numFixableVulns int32
		var fixedByProvided bool
		for _, c := range i.GetScan().GetComponents() {
			numVulns += int32(len(c.GetVulns()))
			for _, v := range c.GetVulns() {
				if v.GetSetFixedBy() != nil {
					fixedByProvided = true
					if v.GetFixedBy() != "" {
						numFixableVulns++
					}
				}
			}
		}
		listImage.SetCves = &storage.ListImage_Cves{
			Cves: numVulns,
		}
		if numVulns == 0 || fixedByProvided {
			listImage.SetFixable = &storage.ListImage_FixableCves{
				FixableCves: numFixableVulns,
			}
		}
	}
	return listImage
}
