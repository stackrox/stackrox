package saml

import (
	"encoding/xml"

	"github.com/russellhaering/gosaml2/types"
)

var (
	// entitiesDescriptorName is the XML name of the EntitiesDescriptor element returned by some
	// IdPs (e.g., Keycloak) in its SAML metadata.
	entitiesDescriptorName = xml.Name{
		Space: "urn:oasis:names:tc:SAML:2.0:metadata",
		Local: "EntitiesDescriptor",
	}
)

// entitiesDescriptor is the relevant portion of the EntitiesDescriptor element (not part of the SAML library
// that we are using).
type entitiesDescriptor struct {
	XMLName           xml.Name                 `xml:"urn:oasis:names:tc:SAML:2.0:metadata EntitiesDescriptor"`
	EntityDescriptors []types.EntityDescriptor `xml:"EntityDescriptor"`
}

// entityDescriptor is a wrapper around an EntityDescriptor slice that can be parsed from either a
// single EntityDescriptor element (resulting in a one-element slice), or an EntitiesDescriptor
// element.
type entityDescriptors []types.EntityDescriptor

func (d *entityDescriptors) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	if start.Name == entitiesDescriptorName {
		var pluralDesc entitiesDescriptor
		if err := dec.DecodeElement(&pluralDesc, &start); err != nil {
			return err
		}
		*d = pluralDesc.EntityDescriptors
		return nil
	}
	var singleDesc types.EntityDescriptor
	if err := dec.DecodeElement(&singleDesc, &start); err != nil {
		return err
	}
	*d = []types.EntityDescriptor{singleDesc}
	return nil
}
