package scannerv4

import (
	"context"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	s4ClientMocks "github.com/stackrox/rox/pkg/scannerv4/client/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetNodeVulnerabilityReportCPECompatibility(t *testing.T) {
	const fullWildcard = "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"
	for caseName, tc := range map[string]struct {
		input string
		want  string
	}{
		"legacy wildcard":     {input: "cpe:2.3:*", want: fullWildcard},
		"full wildcard":       {input: fullWildcard, want: fullWildcard},
		"empty":               {},
		"bound product":       {input: "cpe:2.3:a:redhat:openshift:4.12:*:el8:*:*:*:*:*", want: "cpe:2.3:a:redhat:openshift:4.12:*:el8:*:*:*:*:*"},
		"URI product":         {input: "cpe:/a:redhat:openshift:4.12::el8", want: "cpe:/a:redhat:openshift:4.12::el8"},
		"another short CPE":   {input: "cpe:2.3:*:*", want: "cpe:2.3:*:*"},
		"leading whitespace":  {input: " cpe:2.3:*", want: " cpe:2.3:*"},
		"trailing whitespace": {input: "cpe:2.3:* ", want: "cpe:2.3:* "},
		"trailing newline":    {input: "cpe:2.3:*\n", want: "cpe:2.3:*\n"},
		"invalid":             {input: "not a cpe", want: "not a cpe"},
	} {
		for representation, use := range map[string]struct {
			maps  bool
			lists bool
		}{
			"maps":             {maps: true},
			"deprecated lists": {lists: true},
			"both":             {maps: true, lists: true},
		} {
			t.Run(caseName+"/"+representation, func(t *testing.T) {
				input := nodeIndexWithCPE(tc.input, use.maps, use.lists)
				original := input.CloneVT()
				want := nodeIndexWithCPE(tc.want, use.maps, use.lists)
				response := &v4.VulnerabilityReport{HashId: "test-response"}
				ctrl := gomock.NewController(t)
				mockClient := s4ClientMocks.NewMockScanner(ctrl)
				digest, err := name.NewDigest(mockNodeDigest)
				require.NoError(t, err)
				mockClient.EXPECT().GetVulnerabilities(gomock.Any(), digest, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ name.Digest, contents *v4.Contents, _ ...client.CallOption) (*v4.VulnerabilityReport, error) {
						protoassert.Equal(t, want.GetContents(), contents)
						protoassert.Equal(t, original, input)
						return response, nil
					},
				)
				scanner := &scannerv4{scannerClient: mockClient}
				got, err := scanner.GetNodeVulnerabilityReport(&storage.Node{Name: "test-node"}, input)
				require.NoError(t, err)
				assert.Same(t, response, got)
				protoassert.Equal(t, original, input)
			})
		}
	}
}

func TestNormalizeNodeIndexCPEsNilValues(t *testing.T) {
	for caseName, report := range map[string]*v4.IndexReport{
		"nil report":     nil,
		"nil contents":   {},
		"empty contents": {Contents: &v4.Contents{}},
		"nil elements": {Contents: &v4.Contents{
			Packages:                map[string]*v4.Package{"nil": nil},
			PackagesDEPRECATED:      []*v4.Package{nil},
			Repositories:            map[string]*v4.Repository{"nil": nil},
			RepositoriesDEPRECATED:  []*v4.Repository{nil},
			Distributions:           map[string]*v4.Distribution{"nil": nil},
			DistributionsDEPRECATED: []*v4.Distribution{nil},
		}},
	} {
		t.Run(caseName, func(t *testing.T) {
			original := report.CloneVT()
			protoassert.Equal(t, original, normalizeNodeIndexCPEs(report))
			protoassert.Equal(t, original, report)
		})
	}
}

func TestNormalizeNodeIndexCPEsSourceOnly(t *testing.T) {
	const fullWildcard = "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"
	input := nodeIndexWithCPE("", true, true)
	input.Contents.Packages["600"].Source.Cpe = "cpe:2.3:*"
	input.Contents.PackagesDEPRECATED[0].Source.Cpe = "cpe:2.3:*"
	contents := normalizeNodeIndexCPEs(input).GetContents()
	assert.Empty(t, contents.GetPackages()["600"].GetCpe())
	assert.Empty(t, contents.GetPackagesDEPRECATED()[0].GetCpe())
	assert.Equal(t, fullWildcard, contents.GetPackages()["600"].GetSource().GetCpe())
	assert.Equal(t, fullWildcard, contents.GetPackagesDEPRECATED()[0].GetSource().GetCpe())
	assert.Equal(t, "cpe:2.3:*", input.GetContents().GetPackages()["600"].GetSource().GetCpe())
	assert.Equal(t, "cpe:2.3:*", input.GetContents().GetPackagesDEPRECATED()[0].GetSource().GetCpe())
}

func TestGetVirtualMachineScanDoesNotNormalizeNodeCPEs(t *testing.T) {
	input := nodeIndexWithCPE("cpe:2.3:*", true, true)
	original := input.CloneVT()
	ctrl := gomock.NewController(t)
	mockClient := s4ClientMocks.NewMockScanner(ctrl)
	digest, err := name.NewDigest(mockVirtualMachineDigest)
	require.NoError(t, err)
	mockClient.EXPECT().GetVulnerabilities(gomock.Any(), digest, gomock.Eq(original.GetContents())).
		Return(&v4.VulnerabilityReport{}, nil)
	scanner := &scannerv4{scannerClient: mockClient}
	_, err = scanner.GetVirtualMachineScan(&storage.VirtualMachine{Name: "test-vm"}, input)
	require.NoError(t, err)
	protoassert.Equal(t, original, input)
}

func nodeIndexWithCPE(cpe string, useMaps, useLists bool) *v4.IndexReport {
	pkg := &v4.Package{
		Id: "600", Name: "rhcos", Kind: "binary", Version: "10.2.20261108-0", Cpe: cpe,
		Source: &v4.Package{Id: "600", Name: "rhcos", Kind: "source", Cpe: cpe},
	}
	repo := &v4.Repository{Id: "600", Name: "rhcos-repo", Cpe: cpe}
	dist := &v4.Distribution{Id: "600", Did: "rhel", Cpe: cpe}
	contents := &v4.Contents{
		Environments: map[string]*v4.Environment_List{
			"600": {Environments: []*v4.Environment{{DistributionId: "600", RepositoryIds: []string{"600"}}}},
		},
	}
	if useMaps {
		contents.Packages = map[string]*v4.Package{"600": pkg}
		contents.Repositories = map[string]*v4.Repository{"600": repo}
		contents.Distributions = map[string]*v4.Distribution{"600": dist}
	}
	if useLists {
		contents.PackagesDEPRECATED = []*v4.Package{pkg.CloneVT()}
		contents.RepositoriesDEPRECATED = []*v4.Repository{repo.CloneVT()}
		contents.DistributionsDEPRECATED = []*v4.Distribution{dist.CloneVT()}
	}
	return &v4.IndexReport{HashId: "test-index", Contents: contents}
}
