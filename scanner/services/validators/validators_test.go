package validators

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stretchr/testify/assert"
)

func Test_validateGetVulnerabilitiesRequest(t *testing.T) {
	// Type to set options of GetVulnerabilitiesRequest
	type opts func(*v4.GetVulnerabilitiesRequest)
	// Set content to nil.
	nilContent := func(r *v4.GetVulnerabilitiesRequest) {
		r.ClearContents()
	}
	// Ensure contents exist and apply other options.
	withContent := func(o ...opts) opts {
		return func(r *v4.GetVulnerabilitiesRequest) {
			if r.GetContents() == nil {
				r.SetContents(&v4.Contents{})
			}
			for _, opt := range o {
				opt(r)
			}
		}
	}
	// Add stuff to the properties of contents.
	addPackage := func(p *v4.Package) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.GetContents().GetPackages()[p.GetId()] = p
		})
	}
	addPackageDEPRECATED := func(p *v4.Package) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.GetContents().SetPackagesDEPRECATED(append(r.GetContents().GetPackagesDEPRECATED(), p))
		})
	}
	addDistribution := func(d *v4.Distribution) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.GetContents().GetDistributions()[d.GetId()] = d
		})
	}
	addDistributionDEPRECATED := func(d *v4.Distribution) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.GetContents().SetDistributionsDEPRECATED(append(r.GetContents().GetDistributionsDEPRECATED(), d))
		})
	}
	addRepository := func(d *v4.Repository) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.GetContents().GetRepositories()[d.GetId()] = d
		})
	}
	addRepositoryDEPRECATED := func(d *v4.Repository) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.GetContents().SetRepositoriesDEPRECATED(append(r.GetContents().GetRepositoriesDEPRECATED(), d))
		})
	}
	addEnvironment := func(pkgId string, d *v4.Environment) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			if r.GetContents().GetEnvironments() == nil {
				r.GetContents().SetEnvironments(make(map[string]*v4.Environment_List))
			}
			envs, ok := r.GetContents().GetEnvironments()[pkgId]
			if !ok {
				envs = &v4.Environment_List{}
				r.GetContents().GetEnvironments()[pkgId] = envs
			}
			envs.SetEnvironments(append(envs.GetEnvironments(), d))
		})
	}
	addEnvironmentDEPRECATED := func(pkgId string, d *v4.Environment) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			if r.GetContents().GetEnvironmentsDEPRECATED() == nil {
				r.GetContents().SetEnvironmentsDEPRECATED(make(map[string]*v4.Environment_List))
			}
			envs, ok := r.GetContents().GetEnvironmentsDEPRECATED()[pkgId]
			if !ok {
				envs = &v4.Environment_List{}
				r.GetContents().GetEnvironmentsDEPRECATED()[pkgId] = envs
			}
			envs.SetEnvironments(append(envs.GetEnvironments(), d))
		})
	}
	// Test cases.
	tests := map[string]struct {
		arg     *v4.GetVulnerabilitiesRequest
		argOpts []opts
		wantErr string
	}{
		"when request is nil": {
			wantErr: "empty request",
		},
		"when the hash id is invalid": {
			wantErr: "invalid hash id",
			arg:     v4.GetVulnerabilitiesRequest_builder{HashId: "something not expected"}.Build(),
		},
		"when content is empty": {
			argOpts: []opts{nilContent},
		},
		"when one of the packages is empty": {
			wantErr: "Contents.Packages element \"\" is empty",
			argOpts: []opts{
				addPackage(v4.Package_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addPackage(nil),
				addPackageDEPRECATED(v4.Package_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addPackageDEPRECATED(nil),
			},
		},
		"when one of the packages doesn't have id": {
			wantErr: `Contents.Packages element "": ID is empty`,
			argOpts: []opts{
				addPackage(v4.Package_builder{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addPackage(v4.Package_builder{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addPackageDEPRECATED(v4.Package_builder{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addPackageDEPRECATED(v4.Package_builder{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
			},
		},
		"when one of the packages has an invalid CPE": {
			wantErr: `Contents.Packages element "bar": invalid CPE: cpe: string does not appear to be a bound WFN: "something weird"`,
			argOpts: []opts{
				addPackage(v4.Package_builder{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addPackage(v4.Package_builder{Id: "bar", Cpe: "something weird"}.Build()),
				addPackageDEPRECATED(v4.Package_builder{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addPackageDEPRECATED(v4.Package_builder{Id: "bar", Cpe: "something weird"}.Build()),
			},
		},
		"when a package does not have a source package": {
			argOpts: []opts{
				addPackage(v4.Package_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addPackageDEPRECATED(v4.Package_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
			},
		},
		"when a package has a source package with another source package": {
			wantErr: `Contents.Packages element "foo": package ID="foo" has a source with a source`,
			argOpts: []opts{
				addPackage(v4.Package_builder{
					Id:     "foo",
					Cpe:    "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					Source: v4.Package_builder{Id: "foobar", Source: &v4.Package{}}.Build(),
				}.Build()),
				addPackageDEPRECATED(v4.Package_builder{
					Id:     "foo",
					Cpe:    "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					Source: v4.Package_builder{Id: "foobar", Source: &v4.Package{}}.Build(),
				}.Build()),
			},
		},
		"when one of the distributions is empty": {
			wantErr: "Contents.Distributions element \"\" is empty",
			argOpts: []opts{
				addDistribution(v4.Distribution_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addDistribution(nil),
				addDistributionDEPRECATED(v4.Distribution_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addDistributionDEPRECATED(nil),
			},
		},
		"when one of the distributions doesn't have id": {
			wantErr: "Contents.Distributions element \"\": ID is empty",
			argOpts: []opts{
				addDistribution(v4.Distribution_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addDistribution(v4.Distribution_builder{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addDistributionDEPRECATED(v4.Distribution_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addDistributionDEPRECATED(v4.Distribution_builder{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
			},
		},
		"when one of the distributions has an invalid CPE": {
			wantErr: `Contents.Distributions element "bar": invalid CPE: cpe: string does not appear to be a bound WFN: "something weird"`,
			argOpts: []opts{
				addDistribution(v4.Distribution_builder{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addDistribution(v4.Distribution_builder{Id: "bar", Cpe: "something weird"}.Build()),
				addDistributionDEPRECATED(v4.Distribution_builder{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addDistributionDEPRECATED(v4.Distribution_builder{Id: "bar", Cpe: "something weird"}.Build()),
			},
		},
		"when one of the repositories is empty": {
			wantErr: "Contents.Repositories element \"\" is empty",
			argOpts: []opts{
				addRepository(v4.Repository_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addRepository(nil),
				addRepositoryDEPRECATED(v4.Repository_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addRepositoryDEPRECATED(nil),
			},
		},
		"when one of the repositories doesn't have an id": {
			wantErr: "Contents.Repositories element \"\": ID is empty",
			argOpts: []opts{
				addRepository(v4.Repository_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addRepository(v4.Repository_builder{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addRepositoryDEPRECATED(v4.Repository_builder{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addRepositoryDEPRECATED(v4.Repository_builder{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
			},
		},
		"when one of the repositories has an invalid CPE": {
			wantErr: `Contents.Repositories element "bar": invalid CPE: cpe: string does not appear to be a bound WFN: "something weird"`,
			argOpts: []opts{
				addRepository(v4.Repository_builder{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addRepository(v4.Repository_builder{Id: "bar", Cpe: "something weird"}.Build()),
				addRepositoryDEPRECATED(v4.Repository_builder{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}.Build()),
				addRepositoryDEPRECATED(v4.Repository_builder{Id: "bar", Cpe: "something weird"}.Build()),
			},
		},
		"when one of the envs reference an invalid valid layer digest": {
			argOpts: []opts{
				addEnvironment("foo", v4.Environment_builder{
					IntroducedIn: "sha256:0f2e5032c45d68a9585f035970708802bbc7e2688e1552a4c3d8a7c38c3090c3",
				}.Build()),
				addEnvironment("bar", v4.Environment_builder{
					IntroducedIn: "sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e",
				}.Build()),
				addEnvironmentDEPRECATED("foo", v4.Environment_builder{
					IntroducedIn: "sha256:0f2e5032c45d68a9585f035970708802bbc7e2688e1552a4c3d8a7c38c3090c3",
				}.Build()),
				addEnvironmentDEPRECATED("bar", v4.Environment_builder{
					IntroducedIn: "sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e",
				}.Build()),
			},
		},
		"when all the envs reference valid layer digests": {
			argOpts: []opts{
				addEnvironment("foo", v4.Environment_builder{
					IntroducedIn: "sha256:0f2e5032c45d68a9585f035970708802bbc7e2688e1552a4c3d8a7c38c3090c3",
				}.Build()),
				addEnvironment("bar", v4.Environment_builder{
					IntroducedIn: "unexpected layer digest",
				}.Build()),
				addEnvironmentDEPRECATED("foo", v4.Environment_builder{
					IntroducedIn: "sha256:0f2e5032c45d68a9585f035970708802bbc7e2688e1552a4c3d8a7c38c3090c3",
				}.Build()),
				addEnvironmentDEPRECATED("bar", v4.Environment_builder{
					IntroducedIn: "unexpected layer digest",
				}.Build()),
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// If not set but with options, use default.
			if tt.arg == nil && len(tt.argOpts) > 0 {
				contents := &v4.Contents{}
				contents.SetPackages(map[string]*v4.Package{})
				contents.SetDistributions(map[string]*v4.Distribution{})
				contents.SetRepositories(map[string]*v4.Repository{})
				contents.SetEnvironments(map[string]*v4.Environment_List{})
				gvr := &v4.GetVulnerabilitiesRequest{}
				gvr.SetHashId("/v4/containerimage/foobar")
				gvr.SetContents(contents)
				tt.arg = gvr
			}
			for _, o := range tt.argOpts {
				o(tt.arg)
			}
			err := ValidateGetVulnerabilitiesRequest(tt.arg)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_validateGetSBOMRequest(t *testing.T) {
	tests := map[string]struct {
		req     *v4.GetSBOMRequest
		wantErr string
	}{
		"error on nil req": {
			wantErr: "empty request",
		},
		"error on no id": {
			req:     &v4.GetSBOMRequest{},
			wantErr: "id is required",
		},
		"error on no name": {
			req:     v4.GetSBOMRequest_builder{Id: "id"}.Build(),
			wantErr: "name is required",
		},
		"error on no uri": {
			req:     v4.GetSBOMRequest_builder{Id: "id", Name: "name"}.Build(),
			wantErr: "uri is required",
		},
		"error on empty contentx": {
			req: v4.GetSBOMRequest_builder{Id: "id", Name: "name", Uri: "uri"}.Build(),

			wantErr: "contents are required",
		},
		// This test ensures that the validation logic is executed on the request contents.
		// We do not exercise every possible path where contents are invalid since
		// those paths are already tested as part of Test_validateGetVulnerabilitiesRequest.
		"error on invalid contents": {
			req: v4.GetSBOMRequest_builder{
				Id:   "id",
				Name: "name",
				Uri:  "uri",
				Contents: v4.Contents_builder{
					Packages: map[string]*v4.Package{
						"": {},
					},
				}.Build(),
			}.Build(),
			wantErr: "Contents.Packages element \"\": ID is empty",
		},
		"no error on valid req": {
			req: v4.GetSBOMRequest_builder{
				Id:       "id",
				Name:     "name",
				Uri:      "uri",
				Contents: &v4.Contents{},
			}.Build(),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateGetSBOMRequest(tt.req)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
