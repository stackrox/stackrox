package flags

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReadConfig(t *testing.T) {

	testCases := []struct {
		configFile    string
		configVersion string
		instances     Instance
		users         AuthInfo
		contexts      Context
		currContext   string
		err           string
	}{
		{
			configFile:    "./testdata/test_config1.yaml",
			configVersion: "v1",
			instances: Instance{
				InstanceName:          "endpoint",
				Endpoint:              "hello",
				CaCertificatePath:     "",
				CaCertificate:         "",
				Plaintext:             "",
				DirectGRPC:            false,
				ForceHTTP1:            false,
				Insecure:              false,
				InsecureSkipTLSVerify: false,
			},
			users: AuthInfo{
				Username:         "test1",
				Password:         "",
				ApiToken:         "",
				ApiTokenFilePath: "",
			},
			contexts: Context{
				AuthInfo: "developer",
				Instance: "endpoint",
			},
			currContext: "",
		},
		{
			configFile:    "./testdata/test_config2.yaml",
			configVersion: "1.0",
			instances: Instance{
				InstanceName:          "test-instance",
				Endpoint:              "https://stackrox.example.com",
				CaCertificatePath:     "/path/to/ca.crt",
				CaCertificate:         "certificate-content",
				Plaintext:             "some-text",
				DirectGRPC:            true,
				ForceHTTP1:            false,
				Insecure:              false,
				InsecureSkipTLSVerify: true,
			},
			users: AuthInfo{
				Username:         "test-user",
				Password:         "secure-password",
				ApiToken:         "12345abcde",
				ApiTokenFilePath: "/path/to/api/token",
			},
			contexts: Context{
				AuthInfo: "test-user",
				Instance: "test-instance",
			},
			currContext: "test-instance-context",
		},
		{
			configFile:    "./testdata/test_config3.yaml",
			configVersion: "1.0",
			instances: Instance{
				InstanceName:          "minimal-instance",
				Endpoint:              "https://minimal.stackrox.example.com",
				CaCertificate:         "",
				CaCertificatePath:     "",
				Plaintext:             "",
				DirectGRPC:            false,
				ForceHTTP1:            false,
				Insecure:              false,
				InsecureSkipTLSVerify: false,
			},
			users: AuthInfo{
				Username:         "minimal-user",
				Password:         "",
				ApiToken:         "token-only",
				ApiTokenFilePath: "",
			},
			contexts: Context{
				AuthInfo: "minimal-user",
				Instance: "minimal-instance",
			},
			currContext: "minimal-instance-context",
		},
		{
			configFile:    "./testdata/test_config4.yaml",
			configVersion: "2.0",
			instances: Instance{
				InstanceName:          "insecure-instance",
				Endpoint:              "https://insecure.stackrox.example.com",
				CaCertificate:         "",
				CaCertificatePath:     "",
				Plaintext:             "",
				DirectGRPC:            false,
				ForceHTTP1:            false,
				Insecure:              true,
				InsecureSkipTLSVerify: true,
			},
			users: AuthInfo{
				Username:         "insecure-user",
				Password:         "",
				ApiToken:         "",
				ApiTokenFilePath: "/path/to/insecure/token",
			},
			contexts: Context{
				AuthInfo: "insecure-user",
				Instance: "insecure-instance",
			},
			currContext: "insecure-instance-context",
		},
		{
			configFile:    "./testdata/test_config5.yaml",
			configVersion: "2.1",
			instances: Instance{
				InstanceName:          "plaintext-grpc-instance",
				Endpoint:              "https://grpc.stackrox.example.com",
				CaCertificate:         "",
				CaCertificatePath:     "",
				Plaintext:             "plaintext-content",
				DirectGRPC:            true,
				ForceHTTP1:            false,
				Insecure:              false,
				InsecureSkipTLSVerify: false,
			},
			users: AuthInfo{
				Username:         "grpc-user",
				Password:         "grpc-secure-password",
				ApiToken:         "",
				ApiTokenFilePath: "",
			},
			contexts: Context{
				AuthInfo: "grpc-user",
				Instance: "plaintext-grpc-instance",
			},
			currContext: "plaintext-grpc-context",
		},
		{
			configFile:    "./testdata/test_config6.yaml",
			configVersion: "3.0",
			instances: Instance{
				InstanceName:          "full-instance",
				Endpoint:              "https://full.stackrox.example.com",
				CaCertificatePath:     "/path/to/full/ca.crt",
				CaCertificate:         "full-certificate-content",
				Plaintext:             "full-plaintext",
				DirectGRPC:            false,
				ForceHTTP1:            false,
				Insecure:              false,
				InsecureSkipTLSVerify: true,
			},
			users: AuthInfo{
				Username:         "full-user",
				Password:         "full-password",
				ApiToken:         "full-api-token",
				ApiTokenFilePath: "/path/to/full/api/token",
			},
			contexts: Context{
				AuthInfo: "full-user",
				Instance: "full-instance",
			},
			currContext: "full-instance-context",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.configFile, func(t *testing.T) {
			instance, err := readConfig(tc.configFile)

			if tc.err == "" {
				require.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.err)
				errors.Wrapf(err, "got: %v instead", err)
			}

			version := tc.configVersion
			instances := tc.instances
			users := tc.users
			contexts := tc.contexts
			currContext := tc.currContext

			fmt.Printf("This is the configuration object: %v", instance)
			assert.NotEmpty(t, instance)

			assert.Equal(t, version, instance.Version)
			assert.Equal(t, instances, *instance.Instances)
			assert.Equal(t, users, *instance.AuthInfo)
			assert.Equal(t, contexts, *instance.Contexts)
			assert.Equal(t, currContext, instance.CurrContext)
		})
	}
}
