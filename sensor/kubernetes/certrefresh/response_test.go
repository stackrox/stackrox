package certrefresh

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func createCertificatesSet() *storage.TypedServiceCertificateSet {
	return &storage.TypedServiceCertificateSet{
		CaPem: []byte("ca_cert_pem"),
		ServiceCerts: []*storage.TypedServiceCertificate{
			{
				ServiceType: storage.ServiceType_SCANNER_SERVICE,
				Cert: &storage.ServiceCertificate{
					CertPem: []byte("scanner_cert_pem"),
					KeyPem:  []byte("scanner_key_pem"),
				},
			},
			{
				ServiceType: storage.ServiceType_SENSOR_SERVICE,
				Cert: &storage.ServiceCertificate{
					CertPem: []byte("sensor_cert_pem"),
					KeyPem:  []byte("sensor_key_pem"),
				},
			},
		},
	}
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
			input: &central.IssueSecuredClusterCertsResponse{
				RequestId: "12345",
				Response: &central.IssueSecuredClusterCertsResponse_Error{
					Error: &central.SecuredClusterCertsIssueError{
						Message: errorMessage,
					},
				},
			},
			expectedResult: &Response{
				RequestId:    "12345",
				ErrorMessage: &errorMessage,
				Certificates: nil,
			},
		},
		{
			name: "Response with certificates",
			input: &central.IssueSecuredClusterCertsResponse{
				RequestId: "67890",
				Response: &central.IssueSecuredClusterCertsResponse_Certificates{
					Certificates: certificatesSet,
				},
			},
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
