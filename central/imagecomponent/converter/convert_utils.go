package converter

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/imagecomponent"
	"github.com/stackrox/rox/generated/storage"
)

// ProtoImageComponentToEmbeddedImageScanComponent converts a *storage.ImageComponent proto object to *storage.EmbeddedImageScanComponent proto object
// `vulns` and `layer_index` does not get set.
func ProtoImageComponentToEmbeddedImageScanComponent(component *storage.ImageComponent) *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:     component.GetName(),
		Version:  component.GetVersion(),
		License:  proto.Clone(component.GetLicense()).(*storage.License),
		Priority: component.GetPriority(),
		Source:   component.GetSource(),
	}
}

// EmbeddedImageScanComponentToProtoImageComponent converts a *storage.EmbeddedImageScanComponent proto object to *storage.ImageComponent proto object
// `vulns` and `layer_index` does not get set.
func EmbeddedImageScanComponentToProtoImageComponent(component *storage.EmbeddedImageScanComponent) *storage.ImageComponent {
	return &storage.ImageComponent{
		Id:       imagecomponent.ComponentID{Name: component.GetName(), Version: component.GetVersion()}.ToString(),
		Name:     component.GetName(),
		Version:  component.GetVersion(),
		License:  proto.Clone(component.GetLicense()).(*storage.License),
		Priority: component.GetPriority(),
		Source:   component.GetSource(),
	}
}
