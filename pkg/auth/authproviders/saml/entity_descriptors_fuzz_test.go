package saml

import (
	"encoding/xml"
	"testing"
)

// FuzzEntityDescriptorsUnmarshal tests that the custom UnmarshalXML implementation
// for entityDescriptors does not panic when fed arbitrary XML input.
//
// The custom UnmarshalXML handles both EntityDescriptor and EntitiesDescriptor
// XML elements with namespace handling, so we need to ensure it's robust against:
// - Malformed XML
// - Invalid namespace declarations
// - Unexpected element structures
// - Invalid entity IDs or attributes
// - Deeply nested structures
// - Missing required elements
// - Invalid character encodings
func FuzzEntityDescriptorsUnmarshal(f *testing.F) {
	// Seed corpus: valid EntitiesDescriptor XML
	f.Add([]byte(`<?xml version="1.0"?>
<md:EntitiesDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" Name="urn:keycloak">
  <md:EntityDescriptor entityID="http://localhost:8080/auth/realms/master">
    <md:IDPSSODescriptor WantAuthnRequestsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
      <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
    </md:IDPSSODescriptor>
  </md:EntityDescriptor>
</md:EntitiesDescriptor>`))

	// Seed corpus: valid EntityDescriptor XML
	f.Add([]byte(`<?xml version="1.0"?>
<md:EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" entityID="http://localhost:8080/auth/realms/master">
  <md:IDPSSODescriptor WantAuthnRequestsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
  </md:IDPSSODescriptor>
</md:EntityDescriptor>`))

	// Seed corpus: minimal EntitiesDescriptor
	f.Add([]byte(`<EntitiesDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"></EntitiesDescriptor>`))

	// Seed corpus: minimal EntityDescriptor
	f.Add([]byte(`<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="test"></EntityDescriptor>`))

	// Seed corpus: empty XML
	f.Add([]byte(`<?xml version="1.0"?><root/>`))

	// Seed corpus: malformed but interesting edge cases
	f.Add([]byte(`<EntitiesDescriptor><EntityDescriptor/></EntitiesDescriptor>`))
	f.Add([]byte(`<EntityDescriptor entityID=""/>`))
	f.Add([]byte(``))
	f.Add([]byte(`<`))
	f.Add([]byte(`>`))
	f.Add([]byte(`</>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// The fuzzer's goal is to ensure no panics occur, regardless of input.
		// We don't care if unmarshaling succeeds or fails, only that it doesn't panic.
		var descs entityDescriptors
		_ = xml.Unmarshal(data, &descs)
		// If we reach here without panic, the test passes for this input.
	})
}
