package keys

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"reflect"
	"testing"
	"time"
)

func TestGeneratedCertificateAttributes(t *testing.T) {
	t.Parallel()

	pub, _, err := GenerateStackRoxKeyPair()
	if err != nil {
		t.Error(err)
	}
	cert, err := pub.ToX509()
	if err != nil {
		t.Error(err)
	}

	if cert.SerialNumber.Int64() < 0 || cert.SerialNumber.Int64() > 2<<20 {
		t.Errorf("Invalid Serial Number %d", cert.SerialNumber)
	}
	expectedSubject := pkix.Name{Country: []string{"US"}, Organization: []string{"StackRox"}, OrganizationalUnit: []string{"StackRox"}, Locality: []string{"Mountain View"}, Province: []string(nil), StreetAddress: []string(nil), PostalCode: []string(nil), SerialNumber: "", CommonName: "SSO SP Cert", Names: []pkix.AttributeTypeAndValue{pkix.AttributeTypeAndValue{Type: asn1.ObjectIdentifier{2, 5, 4, 6}, Value: "US"}, pkix.AttributeTypeAndValue{Type: asn1.ObjectIdentifier{2, 5, 4, 7}, Value: "Mountain View"}, pkix.AttributeTypeAndValue{Type: asn1.ObjectIdentifier{2, 5, 4, 10}, Value: "StackRox"}, pkix.AttributeTypeAndValue{Type: asn1.ObjectIdentifier{2, 5, 4, 11}, Value: "StackRox"}, pkix.AttributeTypeAndValue{Type: asn1.ObjectIdentifier{2, 5, 4, 3}, Value: "SSO SP Cert"}}, ExtraNames: []pkix.AttributeTypeAndValue(nil)}

	if !reflect.DeepEqual(cert.Subject, expectedSubject) {
		t.Errorf("Invalid Subject: %#v", cert.Subject)
	}
	if cert.NotBefore.After(time.Now()) {
		t.Errorf("Invalid NotBefore time: %v", cert.NotBefore)
	}
	if cert.NotAfter.Before(time.Now().Add(time.Hour * 24 * 365 * 9)) {
		t.Errorf("Invalid NotAfter time: %v", cert.NotAfter)
	}
	if !cert.IsCA {
		t.Error("Cert should be a CA")
	}
	if cert.MaxPathLen > 0 {
		t.Error("Cert should not be used to authenticate paths")
	}
	if len(cert.PermittedDNSDomains) > 0 {
		t.Error("Cert should not be used for specific domains")
	}
	if cert.SignatureAlgorithm != x509.SHA256WithRSA {
		t.Errorf("Unexpected signature algorith: %T", cert.SignatureAlgorithm)
	}

	possibleFQDNs := []string{"launcher.stackrox", "director.stackrox"}
	for _, n := range possibleFQDNs {
		if err = cert.VerifyHostname(n); err != nil {
			t.Errorf("Hostname failed verification: %s", err)
		}
	}
}
