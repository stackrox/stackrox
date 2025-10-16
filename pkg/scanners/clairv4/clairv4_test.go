package clairv4

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/quay/claircore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

var _ types.Registry = (*mockRegistry)(nil)

type mockRegistry struct {
	url string
}

func (m *mockRegistry) Match(_ *storage.ImageName) bool {
	panic("unsupported")
}

func (m *mockRegistry) Metadata(_ *storage.Image) (*storage.ImageMetadata, error) {
	panic("unsupported")
}

func (m *mockRegistry) Test() error {
	panic("unsupported")
}

func (m *mockRegistry) Config(_ context.Context) *types.Config {
	return &types.Config{
		URL: m.url,
	}
}

func (m *mockRegistry) Name() string {
	panic("unsupported")
}

func (m *mockRegistry) HTTPClient() *http.Client {
	return http.DefaultClient
}

var (
	_ http.Handler = (*noopHandler)(nil)
	_ http.Handler = (*mockClair)(nil)
)

type noopHandler struct{}

func (n *noopHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}

type mockClair struct{}

func (m *mockClair) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check existence of manifest.
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, indexReportPath) {
		// Always say the index does not already exist.
		w.WriteHeader(http.StatusNotFound)
	}

	// index new manifest.
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, indexPath) {
		defer utils.IgnoreError(r.Body.Close)

		var m claircore.Manifest
		_ = json.NewDecoder(r.Body).Decode(&m)

		w.WriteHeader(http.StatusCreated)
		ir := &claircore.IndexReport{
			Hash:    m.Hash,
			Success: true,
			Err:     "",
		}
		_ = json.NewEncoder(w).Encode(ir)
	}

	// get vulnerability report.
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, vulnerabilityReportPath) {
		vr := &claircore.VulnerabilityReport{
			Hash: claircore.MustParseDigest(path.Base(r.URL.Path)),
			Distributions: map[string]*claircore.Distribution{
				"rhel": {
					DID:       "rhel",
					VersionID: "8",
				},
			},
			Packages: map[string]*claircore.Package{
				"a": {
					Name:    "a",
					Version: "1.2.3",
				},
				"b": {
					Name:    "b",
					Version: "4.5",
				},
			},
			PackageVulnerabilities: map[string][]string{
				"a": {"CVE-2023-0001", "CVE-2023-0002"},
				"b": {},
			},
			Vulnerabilities: map[string]*claircore.Vulnerability{
				"CVE-2023-0001": {
					Name:               "CVE-2023-0001",
					Description:        "First CVE of 2023",
					Links:              "https://cve.com/CVE-2023-0001 https://somewhereelse.com",
					NormalizedSeverity: claircore.Medium,
				},
				"CVE-2023-0002": {
					Name:               "CVE-2023-0002",
					Description:        "Second CVE of 2023",
					Links:              "https://cve.com/CVE-2023-0002 https://somewhereelse.com",
					NormalizedSeverity: claircore.Critical,
					FixedInVersion:     "1.2.3.4",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(vr)
	}
}

type imageTestCase struct {
	image    *storage.Image
	expected *storage.ImageScan
}

var testImage = imageTestCase{
	image: storage.Image_builder{
		Id: "sha256:e361a57a7406adee653f1dcff660d84f0ca302907747af2a387f67821acfce33",
		Name: storage.ImageName_builder{
			Registry: "quay.io",
			Remote:   "hello/howdy",
			Tag:      "123",
			FullName: "quay.io/hello/howdy:123",
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			LayerShas: []string{
				"sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
				"sha256:87298cc2f31fba73181ea2a9e6ef10dce21ed95e98bdac9c4e1504ea16f486e4",
			},
		}.Build(),
	}.Build(),
	expected: storage.ImageScan_builder{
		OperatingSystem: "rhel:8",
		Components: []*storage.EmbeddedImageScanComponent{
			storage.EmbeddedImageScanComponent_builder{
				Name:    "a",
				Version: "1.2.3",
				Vulns: []*storage.EmbeddedVulnerability{
					storage.EmbeddedVulnerability_builder{
						Cve:               "CVE-2023-0001",
						Summary:           "First CVE of 2023",
						Link:              "https://cve.com/CVE-2023-0001",
						VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
						Severity:          storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
					}.Build(),
					storage.EmbeddedVulnerability_builder{
						Cve:               "CVE-2023-0002",
						Summary:           "Second CVE of 2023",
						Link:              "https://cve.com/CVE-2023-0002",
						VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
						Severity:          storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
						FixedBy:           proto.String("1.2.3.4"),
					}.Build(),
				},
			}.Build(),
			storage.EmbeddedImageScanComponent_builder{
				Name:    "b",
				Version: "4.5",
				Vulns:   []*storage.EmbeddedVulnerability{},
			}.Build(),
		},
	}.Build(),
}

func TestGetScan(t *testing.T) {
	noop := httptest.NewServer(&noopHandler{})
	defer noop.Close()
	registry := &mockRegistry{url: noop.URL}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSet := mocks.NewMockSet(ctrl)
	mockSet.EXPECT().GetRegistryByImage(gomock.Any()).AnyTimes().Return(registry)

	clairServer := httptest.NewServer(&mockClair{})
	defer clairServer.Close()
	cv4c := &storage.ClairV4Config{}
	cv4c.SetEndpoint(clairServer.URL)
	cv4c.SetInsecure(true)
	ii := &storage.ImageIntegration{}
	ii.SetName("Mock Clair v4")
	ii.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY})
	ii.SetClairV4(proto.ValueOrDefault(cv4c))
	clair, err := newScanner(ii, mockSet)
	require.NoError(t, err)

	scan, err := clair.GetScan(testImage.image)
	assert.NoError(t, err)
	expected := testImage.expected
	protoassert.ElementsMatch(t, expected.GetComponents(), scan.GetComponents())
	assert.Equal(t, expected.GetOperatingSystem(), scan.GetOperatingSystem())
}
