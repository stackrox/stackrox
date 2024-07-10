//go:build test_all

package saml

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalEntitiesDescriptor(t *testing.T) {
	const xmlDoc = `
<?xml version="1.0"?>
<md:EntitiesDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" Name="urn:keycloak">
  <md:EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" entityID="http://localhost:8080/auth/realms/master">
    <md:IDPSSODescriptor WantAuthnRequestsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
      <md:KeyDescriptor use="signing">
        <ds:KeyInfo>
          <ds:KeyName>4O8EqjqUHoTr2cRjbKq-dz7zb9Vzf-Vfc0l4ZrGVLJc</ds:KeyName>
          <ds:X509Data>
            <ds:X509Certificate>MIICmzCCAYMCBgF3b2RrqjANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjEwMjA0MjMzMTI4WhcNMzEwMjA0MjMzMzA4WjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCJVrz38z8LfX0ZkfNMvOZPq5rUP6iqp0GXi248i0jseoOO4T+H/KfmsLenQpzfw+iux8Ry64QqYbVQsCiccrQQpmzX+blQFG9ri39pxNarGZgEVsJoaHmQfyP+8j0C931Ko8asEt4PuqUdeg57IWEYiK88sx63Zwi0aKF5v3jjk/7enlf/ah1XAcy8Eu6QMnSZBUaBs77q7tstsjmEoflr+gaBx2rxSdD+g5BHWukqwtPHKwz12hOp36CnUeDU5adcpQD5WtnadpZq+MNoqfuu6t8zrnfj1C8GzER91uUkx+ymIP6sN3XcfDAA1Bo/inGTTkvyzECJTNTY81x283inAgMBAAEwDQYJKoZIhvcNAQELBQADggEBADci9hIORgLwMx/FUv9sjrMAj9WdJ0Bpp3Z9hb4//T5V06GyCbYKfqCpk9JKjr65v4TU/H+0FazpAkSvuDGINPl32o2VhKno+Y4tWMKWwFfzaRKs8R4TevevHaMUMfdZRVKRRdEuATnJ0kpifamXQeTDHCyNj5c1EtmDqHQKHNjOah7UU17Lb5rKD21EqnSz8ycbkUmWmep5bXG8FtzWzPeHefMzmVlMV2mCxJpiVg2PVeVuvD4KOFwX4c3A0ZFCOr4/fV13wi4BLySQqa6A4uL74Ux3bafyPRRsPuXZdEl6eKWFg7DsJGWlQsZrEfe0YM3671Brx7q+fatvGb9gP6M=</ds:X509Certificate>
          </ds:X509Data>
        </ds:KeyInfo>
      </md:KeyDescriptor>
      <md:KeyDescriptor use="signing">
        <ds:KeyInfo>
          <ds:KeyName>VOiJO50J7eWg5BsfOt8bCOZYCkr_aLffcVu6W5vPWQY</ds:KeyName>
          <ds:X509Data>
            <ds:X509Certificate>MIICmzCCAYMCBgF3b2RrkzANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjEwMjA0MjMzMTI4WhcNMzEwMjA0MjMzMzA4WjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCCd14lHyKaDHYnuViK5i2ZxiMJlOaE4lqUuSxy8w+Ztqj9tV27F2y+ZslA8YCy214nNphuSwwVDlVsUcxKzRCCqHL0o9SKiplLUrPp2BloOteRvMPhgQH09oen8F8SNKwvwoCysvurdG5wPar797axfz2uobB2UMfq7HcEq5R9eGnQmMQ/gAhMSdpv6XRZZxFpgtrJYou3lxw8DQyniJwn4JdYnbWbbgSr8c3CijJjsgykG2IkgYl1XwfGbs1ggeibZ/f+dn+uus6CpxL473NQLqgtK9IyMpMexyHb43j4wZTSUQBUOlraGNsUHic8+9a0q6RdXSsQnwCLDhntWcqVAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAERtOaBodUkqwcnwdoM4mXRjJlNcqiG8ZGHQgib/bYfOVZg/1RWEbujnzf6Teqe64BzlBwvNV8ixVv11FzBGdTkxqbHa1WPp8xHpU3g/Qeu4vS31mHbPMPpjyuhaJfqIwq/doVUdvFDSXqvq1EltAxXK01WSReru4B+TXsIPDS+etjCBaCB8nbstxLKLbe2NqgzHoFEeImS1SxxQRdFZNksUAgQPjtEsFn7/TdkwrmF0I2Bz40LTpbW79+X4L5TX6cYmVNX5YCgLASR729tlBS4woaYG860+9x1M67ZMRi+HEXwuG9d75mk28BRehMD44gkkV5Dw7iZGAnjV7W7DrMU=</ds:X509Certificate>
          </ds:X509Data>
        </ds:KeyInfo>
      </md:KeyDescriptor>
      <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
      <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
      <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:persistent</md:NameIDFormat>
      <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</md:NameIDFormat>
      <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified</md:NameIDFormat>
      <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
      <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
      <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
      <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:SOAP" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
    </md:IDPSSODescriptor>
  </md:EntityDescriptor>
</md:EntitiesDescriptor>`

	var descs entityDescriptors
	require.NoError(t, xml.Unmarshal([]byte(xmlDoc), &descs))

	require.Len(t, descs, 1)
	assert.Equal(t, "http://localhost:8080/auth/realms/master", descs[0].EntityID)
	assert.Equal(t, "http://localhost:8080/auth/realms/master/protocol/saml", descs[0].IDPSSODescriptor.SingleSignOnServices[0].Location)
}

func TestUnmarshalEntityDescriptor(t *testing.T) {
	const xmlDoc = `
<?xml version="1.0"?>
<md:EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" entityID="http://localhost:8080/auth/realms/master">
  <md:IDPSSODescriptor WantAuthnRequestsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo>
        <ds:KeyName>4O8EqjqUHoTr2cRjbKq-dz7zb9Vzf-Vfc0l4ZrGVLJc</ds:KeyName>
        <ds:X509Data>
          <ds:X509Certificate>MIICmzCCAYMCBgF3b2RrqjANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjEwMjA0MjMzMTI4WhcNMzEwMjA0MjMzMzA4WjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCJVrz38z8LfX0ZkfNMvOZPq5rUP6iqp0GXi248i0jseoOO4T+H/KfmsLenQpzfw+iux8Ry64QqYbVQsCiccrQQpmzX+blQFG9ri39pxNarGZgEVsJoaHmQfyP+8j0C931Ko8asEt4PuqUdeg57IWEYiK88sx63Zwi0aKF5v3jjk/7enlf/ah1XAcy8Eu6QMnSZBUaBs77q7tstsjmEoflr+gaBx2rxSdD+g5BHWukqwtPHKwz12hOp36CnUeDU5adcpQD5WtnadpZq+MNoqfuu6t8zrnfj1C8GzER91uUkx+ymIP6sN3XcfDAA1Bo/inGTTkvyzECJTNTY81x283inAgMBAAEwDQYJKoZIhvcNAQELBQADggEBADci9hIORgLwMx/FUv9sjrMAj9WdJ0Bpp3Z9hb4//T5V06GyCbYKfqCpk9JKjr65v4TU/H+0FazpAkSvuDGINPl32o2VhKno+Y4tWMKWwFfzaRKs8R4TevevHaMUMfdZRVKRRdEuATnJ0kpifamXQeTDHCyNj5c1EtmDqHQKHNjOah7UU17Lb5rKD21EqnSz8ycbkUmWmep5bXG8FtzWzPeHefMzmVlMV2mCxJpiVg2PVeVuvD4KOFwX4c3A0ZFCOr4/fV13wi4BLySQqa6A4uL74Ux3bafyPRRsPuXZdEl6eKWFg7DsJGWlQsZrEfe0YM3671Brx7q+fatvGb9gP6M=</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo>
        <ds:KeyName>VOiJO50J7eWg5BsfOt8bCOZYCkr_aLffcVu6W5vPWQY</ds:KeyName>
        <ds:X509Data>
          <ds:X509Certificate>MIICmzCCAYMCBgF3b2RrkzANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjEwMjA0MjMzMTI4WhcNMzEwMjA0MjMzMzA4WjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCCd14lHyKaDHYnuViK5i2ZxiMJlOaE4lqUuSxy8w+Ztqj9tV27F2y+ZslA8YCy214nNphuSwwVDlVsUcxKzRCCqHL0o9SKiplLUrPp2BloOteRvMPhgQH09oen8F8SNKwvwoCysvurdG5wPar797axfz2uobB2UMfq7HcEq5R9eGnQmMQ/gAhMSdpv6XRZZxFpgtrJYou3lxw8DQyniJwn4JdYnbWbbgSr8c3CijJjsgykG2IkgYl1XwfGbs1ggeibZ/f+dn+uus6CpxL473NQLqgtK9IyMpMexyHb43j4wZTSUQBUOlraGNsUHic8+9a0q6RdXSsQnwCLDhntWcqVAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAERtOaBodUkqwcnwdoM4mXRjJlNcqiG8ZGHQgib/bYfOVZg/1RWEbujnzf6Teqe64BzlBwvNV8ixVv11FzBGdTkxqbHa1WPp8xHpU3g/Qeu4vS31mHbPMPpjyuhaJfqIwq/doVUdvFDSXqvq1EltAxXK01WSReru4B+TXsIPDS+etjCBaCB8nbstxLKLbe2NqgzHoFEeImS1SxxQRdFZNksUAgQPjtEsFn7/TdkwrmF0I2Bz40LTpbW79+X4L5TX6cYmVNX5YCgLASR729tlBS4woaYG860+9x1M67ZMRi+HEXwuG9d75mk28BRehMD44gkkV5Dw7iZGAnjV7W7DrMU=</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
    <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:persistent</md:NameIDFormat>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</md:NameIDFormat>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified</md:NameIDFormat>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
    <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
    <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
    <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:SOAP" Location="http://localhost:8080/auth/realms/master/protocol/saml"/>
  </md:IDPSSODescriptor>
</md:EntityDescriptor>`

	var descs entityDescriptors
	require.NoError(t, xml.Unmarshal([]byte(xmlDoc), &descs))

	require.Len(t, descs, 1)
	assert.Equal(t, "http://localhost:8080/auth/realms/master", descs[0].EntityID)
	assert.Equal(t, "http://localhost:8080/auth/realms/master/protocol/saml", descs[0].IDPSSODescriptor.SingleSignOnServices[0].Location)
}
