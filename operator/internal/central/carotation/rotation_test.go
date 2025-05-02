package carotation

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DetermineAction(t *testing.T) {
	cases := map[string]struct {
		now                string
		primaryNotBefore   string
		primaryNotAfter    string
		secondaryNotBefore string
		secondaryNotAfter  string
		wantAction         Action
	}{
		"should return no action in first 3/5 of validity": {
			now:              "2026-06-01T00:00:00Z",
			primaryNotBefore: "2025-01-01T00:00:00Z",
			primaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:       NoAction,
		},
		"should add secondary after 3/5 of validity": {
			now:              "2028-01-02T00:00:00Z",
			primaryNotBefore: "2025-01-01T00:00:00Z",
			primaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:       AddSecondary,
		},
		"should promote secondary after 4/5 of validity": {
			now:                "2029-01-02T00:00:00Z",
			primaryNotBefore:   "2025-01-01T00:00:00Z",
			primaryNotAfter:    "2030-01-01T00:00:00Z",
			secondaryNotBefore: "2028-01-01T00:00:00Z",
			secondaryNotAfter:  "2033-01-01T00:00:00Z",
			wantAction:         PromoteSecondary,
		},
		"should delete expired secondary": {
			now:                "2031-01-02T00:00:00Z",
			primaryNotBefore:   "2028-01-01T00:00:00Z",
			primaryNotAfter:    "2033-01-01T00:00:00Z",
			secondaryNotBefore: "2025-01-01T00:00:00Z",
			secondaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:         DeleteSecondary,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, c.now)
			require.NoError(t, err)

			var primary *x509.Certificate
			if c.primaryNotBefore != "" && c.primaryNotAfter != "" {
				primary = generateTestCertWithValidity(t, c.primaryNotBefore, c.primaryNotAfter)
			}

			var secondary *x509.Certificate
			if c.secondaryNotBefore != "" && c.secondaryNotAfter != "" {
				secondary = generateTestCertWithValidity(t, c.secondaryNotBefore, c.secondaryNotAfter)
			}

			action := DetermineAction(primary, secondary, now)
			assert.Equal(t, c.wantAction, action)
		})
	}
}

func TestGenerateCentralTLSData_Rotation(t *testing.T) {
	type testCase struct {
		name            string
		action          Action
		additionalSetup func(t *testing.T, old types.SecretDataMap)
		assert          func(t *testing.T, old, new types.SecretDataMap)
	}

	cases := []testCase{
		{
			name:   "add secondary CA",
			action: AddSecondary,
			assert: func(t *testing.T, old, new types.SecretDataMap) {
				require.Contains(t, new, mtls.SecondaryCACertFileName, "secondary CA cert should be present")
				require.Contains(t, new, mtls.SecondaryCAKeyFileName, "secondary CA key should be present")
				require.Equal(t, old[mtls.CACertFileName], new[mtls.CACertFileName], "primary CA should be unchanged")
			},
		},
		{
			name:   "promote secondary CA",
			action: PromoteSecondary,
			additionalSetup: func(t *testing.T, old types.SecretDataMap) {
				secondary, err := certgen.GenerateCA()
				require.NoError(t, err)
				certgen.AddSecondaryCAToFileMap(old, secondary)
			},
			assert: func(t *testing.T, old, new types.SecretDataMap) {
				require.Contains(t, new, mtls.SecondaryCACertFileName, "secondary CA cert should be present")
				require.Contains(t, new, mtls.SecondaryCAKeyFileName, "secondary CA key should be present")
				require.NotEqual(t, old[mtls.CACertFileName], new[mtls.CACertFileName], "primary CA should have changed")
				require.Equal(t, new[mtls.SecondaryCACertFileName], old[mtls.CACertFileName],
					"secondary CA cert should be the old primary CA cert")
				require.Equal(t, new[mtls.SecondaryCAKeyFileName], old[mtls.CAKeyFileName],
					"secondary CA key should be the old primary CA key")
				require.Equal(t, new[mtls.CACertFileName], old[mtls.SecondaryCACertFileName],
					"primary CA cert should be the old secondary CA cert")
				require.Equal(t, new[mtls.CAKeyFileName], old[mtls.SecondaryCAKeyFileName],
					"primary CA key should be the old secondary CA key")
				require.Contains(t, new, mtls.ServiceCertFileName, "central cert should be present")
				require.Contains(t, new, mtls.ServiceKeyFileName, "central cert should be present")
			},
		},
		{
			name:   "delete secondary CA",
			action: DeleteSecondary,
			additionalSetup: func(t *testing.T, old types.SecretDataMap) {
				secondary, err := certgen.GenerateCA()
				require.NoError(t, err)
				certgen.AddSecondaryCAToFileMap(old, secondary)
			},
			assert: func(t *testing.T, old, new types.SecretDataMap) {
				require.Equal(t, old[mtls.CACertFileName], new[mtls.CACertFileName], "primary CA cert should be unchanged")
				require.Equal(t, old[mtls.CAKeyFileName], new[mtls.CAKeyFileName], "primary CA key should be unchanged")
				require.NotContains(t, new, mtls.SecondaryCACertFileName, "secondary CA cert should be removed")
				require.NotContains(t, new, mtls.SecondaryCAKeyFileName, "secondary CA key should be removed")
			},
		},
		{
			name:   "no rotation action, secondary CA not present",
			action: NoAction,
			assert: func(t *testing.T, old, new types.SecretDataMap) {
				require.Equal(t, old[mtls.CACertFileName], new[mtls.CACertFileName], "primary CA cert should be unchanged")
				require.Equal(t, old[mtls.CAKeyFileName], new[mtls.CAKeyFileName], "primary CA key should be unchanged")
				require.NotContains(t, new, mtls.SecondaryCACertFileName, "secondary CA cert should be present")
				require.NotContains(t, new, mtls.SecondaryCAKeyFileName, "secondary CA key should be present")
			},
		},
		{
			name:   "no rotation action, secondary CA present",
			action: NoAction,
			additionalSetup: func(t *testing.T, old types.SecretDataMap) {
				secondary, err := certgen.GenerateCA()
				require.NoError(t, err)
				certgen.AddSecondaryCAToFileMap(old, secondary)
			},
			assert: func(t *testing.T, old, new types.SecretDataMap) {
				require.Equal(t, old[mtls.CACertFileName], new[mtls.CACertFileName], "primary CA cert should be unchanged")
				require.Equal(t, old[mtls.CAKeyFileName], new[mtls.CAKeyFileName], "primary CA key should be unchanged")
				require.Equal(t, old[mtls.SecondaryCACertFileName], new[mtls.SecondaryCACertFileName], "secondary CA cert should be unchanged")
				require.Equal(t, old[mtls.SecondaryCAKeyFileName], new[mtls.SecondaryCAKeyFileName], "secondary CA key should be unchanged")
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			primary, err := certgen.GenerateCA()
			require.NoError(t, err)
			fileMap := make(types.SecretDataMap)
			certgen.AddCAToFileMap(fileMap, primary)
			err = certgen.IssueCentralCert(fileMap, primary, mtls.WithNamespace("stackrox"))
			require.NoError(t, err)

			if tt.additionalSetup != nil {
				tt.additionalSetup(t, fileMap)
			}

			oldFileMap := maputil.ShallowClone(fileMap)
			err = Handle(tt.action, fileMap)
			require.NoError(t, err)

			tt.assert(t, oldFileMap, fileMap)
		})
	}
}

func generateTestCertWithValidity(t *testing.T, notBeforeStr, notAfterStr string) *x509.Certificate {
	t.Helper()
	notBefore, err := time.Parse(time.RFC3339, notBeforeStr)
	require.NoError(t, err)
	notAfter, err := time.Parse(time.RFC3339, notAfterStr)
	require.NoError(t, err)
	return &x509.Certificate{
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}
}
