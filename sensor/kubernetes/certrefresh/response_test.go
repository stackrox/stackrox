package certrefresh

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func createCertificatesSet() *storage.TypedServiceCertificateSet {
	return storage.TypedServiceCertificateSet_builder{
		CaPem: []byte("ca_cert_pem"),
		ServiceCerts: []*storage.TypedServiceCertificate{
			storage.TypedServiceCertificate_builder{
				ServiceType: storage.ServiceType_SCANNER_SERVICE,
				Cert: storage.ServiceCertificate_builder{
					CertPem: []byte("scanner_cert_pem"),
					KeyPem:  []byte("scanner_key_pem"),
				}.Build(),
			}.Build(),
			storage.TypedServiceCertificate_builder{
				ServiceType: storage.ServiceType_SENSOR_SERVICE,
				Cert: storage.ServiceCertificate_builder{
					CertPem: []byte("sensor_cert_pem"),
					KeyPem:  []byte("sensor_key_pem"),
				}.Build(),
			}.Build(),
		},
	}.Build()
}

func TestConvertSecuredClusterCertsResponse(t *testing.T) {
	errorMessage := "error message"
	certificatesSet := createCertificatesSet()

	tests := []struct {
		name           string
		input          *central.IssueSecuredClusterCertsResponse
		expectedResult *Response
	}{
		{
			name:           "Nil input",
			input:          nil,
			expectedResult: nil,
		},
		{
			name: "Response with error",
			input: central.IssueSecuredClusterCertsResponse_builder{
				RequestId: "12345",
				Error: central.SecuredClusterCertsIssueError_builder{
					Message: errorMessage,
				}.Build(),
			}.Build(),
			expectedResult: &Response{
				RequestId:    "12345",
				ErrorMessage: &errorMessage,
				Certificates: nil,
			},
		},
		{
			name: "Response with certificates",
			input: central.IssueSecuredClusterCertsResponse_builder{
				RequestId:    "67890",
				Certificates: proto.ValueOrDefault(certificatesSet),
			}.Build(),
			expectedResult: &Response{
				RequestId:    "67890",
				ErrorMessage: nil,
				Certificates: certificatesSet,
			},
		},
		{
			name:           "Empty response",
			input:          &central.IssueSecuredClusterCertsResponse{},
			expectedResult: &Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewResponseFromSecuredClusterCerts(tt.input)

			if tt.expectedResult == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expectedResult.RequestId, result.RequestId)
				assert.Equal(t, tt.expectedResult.ErrorMessage, result.ErrorMessage)
				// Must use proto.Equal for the Certificates field
				assert.True(t, proto.Equal(tt.expectedResult.Certificates, result.Certificates), "Certificates should match")
			}
		})
	}
}
