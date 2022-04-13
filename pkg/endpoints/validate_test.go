package endpoints

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestEndpointValidation(t *testing.T) {
	testCases := []struct {
		endpoint    string
		errExpected bool
	}{
		{endpoint: "localhost:8000", errExpected: true},
		{endpoint: "127.0.0.1:8000", errExpected: true},
		{endpoint: "http://localhost:8000", errExpected: true},
		{endpoint: "metadata.google.internal:8000", errExpected: true},
		{endpoint: "169.254.169.254:8000", errExpected: true},
		{endpoint: "https://169.254.169.254:8000", errExpected: true},
		{endpoint: "https://1.1.1.1:8000", errExpected: false},
		{endpoint: "1.1.1.1:8000", errExpected: false},
		{endpoint: "docker.io/localhost", errExpected: false},
	}

	for _, c := range testCases {
		s3Config := storage.S3Config{
			Endpoint:        c.endpoint,
			Bucket:          "buck",
			UseIam:          false,
			AccessKeyId:     "foo",
			SecretAccessKey: "bar",
			Region:          "us-west-2",
			ObjectPrefix:    "",
		}
		err := ValidateEndpoints(&s3Config)
		assert.Equalf(t, c.errExpected, err != nil, "Testcase with endpoint: %s failed", c.endpoint)
	}
}
