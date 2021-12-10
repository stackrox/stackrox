package securedclusterservices

import (
	"fmt"
	"testing"

	helmTest "github.com/stackrox/helmtest/pkg/framework"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/helm/charts"
	metaUtil "github.com/stackrox/rox/pkg/helm/charts/testutils"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestWithHelmtest(t *testing.T) {
	helmImage := image.GetDefaultImage()
	tpl, err := helmImage.GetSecuredClusterServicesChartTemplate()
	require.NoError(t, err, "error retrieving chart template")
	ch, err := tpl.InstantiateAndLoad(metaUtil.MakeMetaValuesForTest(t))
	require.NoError(t, err, "error instantiating chart")

	suite, err := helmTest.NewLoader("testdata/helmtest/chart").LoadSuiteWithFlavour("chart")
	require.NoError(t, err, "failed to load helmtest suite")

	target := &helmTest.Target{
		Chart: ch,
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      "stackrox-secured-cluster-services",
			Namespace: "stackrox",
			IsInstall: true,
		},
	}
	suite.Run(t, target)
}

func TestBundleWithHelmtest(t *testing.T) {
	caCert := []byte(`
-----BEGIN CERTIFICATE-----
MIIB0jCCAXigAwIBAgIUDuyxeeW/uPhPXh1VEkEoy8k5qScwCgYIKoZIzj0EAwIw
RzEnMCUGA1UEAxMeU3RhY2tSb3ggQ2VydGlmaWNhdGUgQXV0aG9yaXR5MRwwGgYD
VQQFExMyNjEwMTE1MzMwMjg0NTM5ODcxMB4XDTIxMTIxMDA5MDQwMFoXDTI2MTIw
OTA5MDQwMFowRzEnMCUGA1UEAxMeU3RhY2tSb3ggQ2VydGlmaWNhdGUgQXV0aG9y
aXR5MRwwGgYDVQQFExMyNjEwMTE1MzMwMjg0NTM5ODcxMFkwEwYHKoZIzj0CAQYI
KoZIzj0DAQcDQgAEPVQ/Oyg9OuGkbLdfzFIkoRq55DI0RCcQyXW4FNzkjyYiheIQ
M40nX8OrqNKl19kQ+2aha5AnfNPz8+xESz/F6qNCMEAwDgYDVR0PAQH/BAQDAgEG
MA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFPqCTvxyQ23AP9zccrKlNZE1HIAo
MAoGCCqGSM49BAMCA0gAMEUCIQDWfGn2/X259pOne8wKikNQV3SIcJOWqb+Qx7Gf
ZgNtGQIgOon9+aGqGUzTONWJM26nEG+9/pnbc0QYHIJzgZIk7Ps=
-----END CERTIFICATE-----
`)
	admissionControllerCert := []byte(`
-----BEGIN CERTIFICATE-----
MIICkjCCAjegAwIBAgIIA9dIbqpbG3YwCgYIKoZIzj0EAwIwRzEnMCUGA1UEAxMe
U3RhY2tSb3ggQ2VydGlmaWNhdGUgQXV0aG9yaXR5MRwwGgYDVQQFExMyNjEwMTE1
MzMwMjg0NTM5ODcxMB4XDTIxMTIxMDA4MTIwMFoXDTIyMTIxMDA5MTIwMFowgYsx
IjAgBgNVBAsMGUFETUlTU0lPTl9DT05UUk9MX1NFUlZJQ0UxSDBGBgNVBAMMP0FE
TUlTU0lPTl9DT05UUk9MX1NFUlZJQ0U6IDIxOWFkMmViLWYxOTUtNDZjNi1iYjgy
LTliOTM4MWVmZDExNTEbMBkGA1UEBRMSMjc2NzY5NTQyMjQ4Mjc0ODA2MFkwEwYH
KoZIzj0CAQYIKoZIzj0DAQcDQgAEUcXy4PQpeNU72NGwxcKGw1r7NUNIzTIBveU/
rhKyQ5DUAgycAwJxWUlNVRU2jy+GWYDrG1+XDgoFPpFBrEOqVqOBxzCBxDAOBgNV
HQ8BAf8EBAMCBaAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMAwGA1Ud
EwEB/wQCMAAwHQYDVR0OBBYEFAF7GFtqB4kUi2QaInVSDvvEPI0/MB8GA1UdIwQY
MBaAFPqCTvxyQ23AP9zccrKlNZE1HIAoMEUGA1UdEQQ+MDyCGmFkbWlzc2lvbi1j
b250cm9sLnN0YWNrcm94gh5hZG1pc3Npb24tY29udHJvbC5zdGFja3JveC5zdmMw
CgYIKoZIzj0EAwIDSQAwRgIhALc0O1ayC4YlPT8t2QJ14hnjOEbQp5oQZANfa9iR
MSddAiEAlQgi9q89EFXbd7LcBfgL6Gm3Re1VRNbO+BA0rB3OThI=
-----END CERTIFICATE-----
`)
	admissionControllerKey := []byte(`
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIG13qSEw6Ic1VNVXwcr5QLkF93mFwdSLFxAlFqdfPoIsoAoGCCqGSM49
AwEHoUQDQgAEUcXy4PQpeNU72NGwxcKGw1r7NUNIzTIBveU/rhKyQ5DUAgycAwJx
WUlNVRU2jy+GWYDrG1+XDgoFPpFBrEOqVg==
-----END EC PRIVATE KEY-----
`)
	collectorCert := []byte(`
-----BEGIN CERTIFICATE-----
MIICdDCCAhmgAwIBAgIJAO79iFeyz9WoMAoGCCqGSM49BAMCMEcxJzAlBgNVBAMT
HlN0YWNrUm94IENlcnRpZmljYXRlIEF1dGhvcml0eTEcMBoGA1UEBRMTMjYxMDEx
NTMzMDI4NDUzOTg3MTAeFw0yMTEyMTAwODEyMDBaFw0yMjEyMTAwOTEyMDBaMH0x
GjAYBgNVBAsMEUNPTExFQ1RPUl9TRVJWSUNFMUAwPgYDVQQDDDdDT0xMRUNUT1Jf
U0VSVklDRTogMjE5YWQyZWItZjE5NS00NmM2LWJiODItOWI5MzgxZWZkMTE1MR0w
GwYDVQQFExQxNzIyMTA3MDQ2MDM3ODE0MjEyMDBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABDh3DcVf1bzJ9Lb21mQcfhl23Vx7IVVQPIJuIBb6qtbSdyhWa73/eK8O
kcdsGo7oRhSx/xx4Fm6VQfc7+EYk2vWjgbcwgbQwDgYDVR0PAQH/BAQDAgWgMB0G
A1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMB0GA1Ud
DgQWBBTYXW+ei03m+YiPGH5uk/js0pw1xTAfBgNVHSMEGDAWgBT6gk78ckNtwD/c
3HKypTWRNRyAKDA1BgNVHREELjAsghJjb2xsZWN0b3Iuc3RhY2tyb3iCFmNvbGxl
Y3Rvci5zdGFja3JveC5zdmMwCgYIKoZIzj0EAwIDSQAwRgIhAPCNNnrFcw2fCGSf
09UOcm6ubWA/dMoefFT7LxnELTbDAiEAw/LMeJVYJgax75FQKu8LZ26irukkK+uT
X0DijvhIVPU=
-----END CERTIFICATE-----
`)
	collectorKey := []byte(`
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIBQE9SO1Smj0Hz8lXQmo/wCQqJiFFOPp1DOXS02vGH8HoAoGCCqGSM49
AwEHoUQDQgAEOHcNxV/VvMn0tvbWZBx+GXbdXHshVVA8gm4gFvqq1tJ3KFZrvf94
rw6Rx2wajuhGFLH/HHgWbpVB9zv4RiTa9Q==
-----END EC PRIVATE KEY-----
`)
	sensorCert := []byte(`
-----BEGIN CERTIFICATE-----
MIICgzCCAiigAwIBAgIIGwUidvXyk3kwCgYIKoZIzj0EAwIwRzEnMCUGA1UEAxMe
U3RhY2tSb3ggQ2VydGlmaWNhdGUgQXV0aG9yaXR5MRwwGgYDVQQFExMyNjEwMTE1
MzMwMjg0NTM5ODcxMB4XDTIxMTIxMDA4MTIwMFoXDTIyMTIxMDA5MTIwMFowdjEX
MBUGA1UECwwOU0VOU09SX1NFUlZJQ0UxPTA7BgNVBAMMNFNFTlNPUl9TRVJWSUNF
OiAyMTlhZDJlYi1mMTk1LTQ2YzYtYmI4Mi05YjkzODFlZmQxMTUxHDAaBgNVBAUT
EzE5NDcwMDAzMDgyMzU0MDgyNDkwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAASp
nT9o6DX7B+wbX7erGTUz2TPQLLgSZlmwGlNdgjHumNSzixK6we2qo5M0RMFzhTqz
xZ4YtIbAzNqRNwrT9O4io4HOMIHLMA4GA1UdDwEB/wQEAwIFoDAdBgNVHSUEFjAU
BggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADAdBgNVHQ4EFgQUuBs9
eWeUoldJ4w7m+wTHYpDsPEUwHwYDVR0jBBgwFoAU+oJO/HJDbcA/3NxysqU1kTUc
gCgwTAYDVR0RBEUwQ4IPc2Vuc29yLnN0YWNrcm94ghNzZW5zb3Iuc3RhY2tyb3gu
c3ZjghtzZW5zb3Itd2ViaG9vay5zdGFja3JveC5zdmMwCgYIKoZIzj0EAwIDSQAw
RgIhAJQqpyNLFCBsG2gl3k7tdsKDuGYjtnNrkfOyfi00JobmAiEAhyHSlGqeyz00
CVkGFtxky4vqF6TfDxn7sIcXuXmosG4=
-----END CERTIFICATE-----
`)
	sensorKey := []byte(`
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIAQWvW0FJ7lw5c10xVvbfvTFByDbprgI9WUGMt1KhsuDoAoGCCqGSM49
AwEHoUQDQgAEqZ0/aOg1+wfsG1+3qxk1M9kz0Cy4EmZZsBpTXYIx7pjUs4sSusHt
qqOTNETBc4U6s8WeGLSGwMzakTcK0/TuIg==
-----END EC PRIVATE KEY-----
`)
	certs := sensor.Certs{
		Files: map[string][]byte{
			"admission-control-cert.pem": admissionControllerCert,
			"admission-control-key.pem":  admissionControllerKey,
			"ca.pem":                     caCert,
			"collector-cert.pem":         collectorCert,
			"collector-key.pem":          collectorKey,
			"sensor-cert.pem":            sensorCert,
			"sensor-key.pem":             sensorKey,
		},
	}
	mainImage, err := utils.GenerateImageFromStringWithDefaultTag("stackrox.io/stackrox/main", version.GetMainVersion())
	require.NoError(t, err, "generating main image name")
	mainImageName := mainImage.GetName()
	collectorImage, err := utils.GenerateImageFromStringWithDefaultTag("stackrox.io/stackrox/collector", version.GetCollectorVersion())
	require.NoError(t, err, "generating collector container image name")
	collectorImageName := collectorImage.GetName()

	centralEndpoint := "central.stackrox:8000"
	collectionMethod := "EBPF"

	chartMetaValues := charts.MetaValues{
		"ClusterName": "test-sensor",
		"ClusterType": "KUBERNETES",

		"ImageRegistry": urlfmt.FormatURL(mainImageName.GetRegistry(), urlfmt.NONE, urlfmt.NoTrailingSlash),
		"MainRegistry":  urlfmt.FormatURL(mainImageName.GetRegistry(), urlfmt.NONE, urlfmt.NoTrailingSlash),
		"ImageRemote":   mainImageName.GetRemote(),
		"ImageTag":      mainImageName.GetTag(),

		"PublicEndpoint":     urlfmt.FormatURL(centralEndpoint, urlfmt.NONE, urlfmt.NoTrailingSlash),
		"AdvertisedEndpoint": urlfmt.FormatURL(env.AdvertisedEndpoint.Setting(), urlfmt.NONE, urlfmt.NoTrailingSlash),

		"CollectorRegistry":        urlfmt.FormatURL(collectorImageName.GetRegistry(), urlfmt.NONE, urlfmt.NoTrailingSlash),
		"CollectorImageRemote":     collectorImageName.GetRemote(),
		"CollectorFullImageTag":    fmt.Sprintf("%s-latest", collectorImageName.GetTag()),
		"CollectorFullImageRemote": collectorImageName.GetRemote(),
		"CollectorSlimImageRemote": collectorImageName.GetRemote(),
		"CollectorSlimImageTag":    fmt.Sprintf("%s-slim", collectorImageName.GetTag()),
		"CollectionMethod":         collectionMethod,

		// Hardcoding RHACS charts repo for now.
		// TODO: fill ChartRepo based on the current image flavor.
		"ChartRepo": defaults.ChartRepo{
			URL: "http://mirror.openshift.com/pub/rhacs/charts",
		},

		"TolerationsEnabled": true,
		"CreateUpgraderSA":   true,

		"EnvVars": map[string]interface{}{},

		"K8sCommand": "kubectl",

		"OfflineMode": env.OfflineModeEnv.BooleanSetting(),

		"SlimCollector": true,

		"KubectlOutput": true,

		"Versions": version.GetAllVersions(),

		"FeatureFlags": make(map[string]string),

		"AdmissionController":              true,
		"AdmissionControlListenOnUpdates":  true,
		"AdmissionControlListenOnEvents":   true,
		"DisableBypass":                    true,
		"TimeoutSeconds":                   10,
		"ScanInline":                       true,
		"AdmissionControllerEnabled":       true,
		"AdmissionControlEnforceOnUpdates": true,
	}

	helmImage := image.GetDefaultImage()
	tpl, err := helmImage.GetSecuredClusterServicesChartTemplate()
	require.NoError(t, err, "error retrieving chart template")
	ch, err := tpl.InstantiateAndLoadWithAdditionalFiles(chartMetaValues, certs.Files)
	require.NoError(t, err, "error instantiating chart")

	suite, err := helmTest.NewLoader("testdata/helmtest/bundle").LoadSuiteWithFlavour("bundle")
	require.NoError(t, err, "failed to load helmtest suite")

	target := &helmTest.Target{
		Chart: ch,
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      "stackrox-secured-cluster-services",
			Namespace: "stackrox",
			IsInstall: true,
		},
	}
	suite.Run(t, target)
}
