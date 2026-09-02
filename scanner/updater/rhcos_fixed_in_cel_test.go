package updater

import (
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/quay/claircore/rhel/vex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRHCOSFixedInVersionCELCompile(t *testing.T) {
	t.Parallel()

	prog, err := vex.CompileFixedInVersionCEL(rhcosFixedInVersionCEL)
	require.NoError(t, err)
	require.NotNil(t, prog)
}

func TestRHCOSFixedInVersionCEL(t *testing.T) {
	t.Parallel()

	prog, err := vex.CompileFixedInVersionCEL(rhcosFixedInVersionCEL)
	require.NoError(t, err)
	require.NotNil(t, prog)

	testcases := []struct {
		name    string
		purl    packageurl.PackageURL
		fixedIn string
		want    string
	}{
		{
			name: "dotted-quad ocp 4.19",
			purl: packageurl.PackageURL{
				Type: packageurl.TypeOCI,
				Name: "rhcos",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"tag": "4.19.9.6.202506252250-0",
				}),
			},
			fixedIn: "4.19.9.6.202506252250-0",
			want:    "9.6.20250625-0",
		},
		{
			name: "dotted-quad ocp 5.0",
			purl: packageurl.PackageURL{
				Type: packageurl.TypeOCI,
				Name: "rhcos-x86_64",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"tag": "5.0.9.8.202607152148-0",
				}),
			},
			fixedIn: "5.0.9.8.202607152148-0",
			want:    "9.8.20260715-0",
		},
		{
			name: "dotted-quad ocp 4.18 left alone",
			purl: packageurl.PackageURL{
				Type: packageurl.TypeOCI,
				Name: "rhcos",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"tag": "4.18.9.6.202501011200-0",
				}),
			},
			fixedIn: "4.18.9.6.202501011200-0",
			want:    "4.18.9.6.202501011200-0",
		},
		{
			name: "nonnumeric ocp minor left alone",
			purl: packageurl.PackageURL{
				Type: packageurl.TypeOCI,
				Name: "rhcos",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"tag": "4.x.9.6.202506252250-0",
				}),
			},
			fixedIn: "4.x.9.6.202506252250-0",
			want:    "4.x.9.6.202506252250-0",
		},
		{
			name: "nonnumeric ocp major left alone",
			purl: packageurl.PackageURL{
				Type: packageurl.TypeOCI,
				Name: "rhcos",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"tag": "a.19.9.6.202506252250-0",
				}),
			},
			fixedIn: "a.19.9.6.202506252250-0",
			want:    "a.19.9.6.202506252250-0",
		},
		{
			name: "compact left alone",
			purl: packageurl.PackageURL{
				Type: packageurl.TypeOCI,
				Name: "rhcos",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"tag": "416.94.202410090804-0",
				}),
			},
			fixedIn: "416.94.202410090804-0",
			want:    "416.94.202410090804-0",
		},
		{
			name: "already rhel-style left alone",
			purl: packageurl.PackageURL{
				Type: packageurl.TypeOCI,
				Name: "rhcos",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"tag": "9.6.20250625-0",
				}),
			},
			fixedIn: "9.6.20250625-0",
			want:    "9.6.20250625-0",
		},
		{
			name: "non-rhcos oci left alone",
			purl: packageurl.PackageURL{
				Type: packageurl.TypeOCI,
				Name: "ubi9",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"tag": "9.4",
				}),
			},
			fixedIn: "9.4",
			want:    "9.4",
		},
		{
			name: "rpm left alone",
			purl: packageurl.PackageURL{
				Type:      packageurl.TypeRPM,
				Namespace: "redhat",
				Name:      "rhcos-kernel",
				Version:   "5.14.0-1.el9",
			},
			fixedIn: "0:5.14.0-1.el9",
			want:    "0:5.14.0-1.el9",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := vex.EvalFixedInVersionCEL(prog, &tc.purl, tc.fixedIn)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRHELVexConfigSetsFixedInVersionCEL(t *testing.T) {
	t.Parallel()

	cfg := &vex.FactoryConfig{}
	require.NoError(t, rhelVexConfig(cfg))
	assert.Equal(t, rhcosFixedInVersionCEL, cfg.FixedInVersionCEL)

	require.NoError(t, rhelVexConfig(&vex.UpdaterConfig{}))
	require.Error(t, rhelVexConfig("bogus"))
}
