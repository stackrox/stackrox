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
		r.Contents = nil
	}
	// Ensure contents exist and apply other options.
	withContent := func(o ...opts) opts {
		return func(r *v4.GetVulnerabilitiesRequest) {
			if r.GetContents() == nil {
				r.Contents = &v4.Contents{}
			}
			for _, opt := range o {
				opt(r)
			}
		}
	}
	// Add stuff to the properties of contents.
	addPackage := func(p *v4.Package) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.Contents.Packages[p.GetId()] = p
		})
	}
	addPackageDEPRECATED := func(p *v4.Package) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.Contents.PackagesDEPRECATED = append(r.Contents.PackagesDEPRECATED, p)
		})
	}
	addDistribution := func(d *v4.Distribution) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.Contents.Distributions[d.GetId()] = d
		})
	}
	addDistributionDEPRECATED := func(d *v4.Distribution) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.Contents.DistributionsDEPRECATED = append(r.Contents.DistributionsDEPRECATED, d)
		})
	}
	addRepository := func(d *v4.Repository) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.Contents.Repositories[d.GetId()] = d
		})
	}
	addRepositoryDEPRECATED := func(d *v4.Repository) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.Contents.RepositoriesDEPRECATED = append(r.Contents.RepositoriesDEPRECATED, d)
		})
	}
	addEnvironment := func(pkgId string, d *v4.Environment) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			if r.Contents.Environments == nil {
				r.Contents.Environments = make(map[string]*v4.Environment_List)
			}
			envs, ok := r.GetContents().GetEnvironments()[pkgId]
			if !ok {
				envs = &v4.Environment_List{}
				r.Contents.Environments[pkgId] = envs
			}
			envs.Environments = append(envs.Environments, d)
		})
	}
	addEnvironmentDEPRECATED := func(pkgId string, d *v4.Environment) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			if r.Contents.EnvironmentsDEPRECATED == nil {
				r.Contents.EnvironmentsDEPRECATED = make(map[string]*v4.Environment_List)
			}
			envs, ok := r.GetContents().GetEnvironmentsDEPRECATED()[pkgId]
			if !ok {
				envs = &v4.Environment_List{}
				r.Contents.EnvironmentsDEPRECATED[pkgId] = envs
			}
			envs.Environments = append(envs.Environments, d)
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
			arg:     &v4.GetVulnerabilitiesRequest{HashId: "something not expected"},
		},
		"when content is empty": {
			argOpts: []opts{nilContent},
		},
		"when one of the packages is empty": {
			wantErr: "Contents.Packages element \"\" is empty",
			argOpts: []opts{
				addPackage(&v4.Package{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addPackage(nil),
				addPackageDEPRECATED(&v4.Package{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addPackageDEPRECATED(nil),
			},
		},
		"when one of the packages doesn't have id": {
			wantErr: `Contents.Packages element "": ID is empty`,
			argOpts: []opts{
				addPackage(&v4.Package{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addPackage(&v4.Package{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addPackageDEPRECATED(&v4.Package{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addPackageDEPRECATED(&v4.Package{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
			},
		},
		"when one of the packages has an invalid CPE": {
			wantErr: `Contents.Packages element "bar": invalid CPE: cpe: string does not appear to be a bound WFN: "something weird"`,
			argOpts: []opts{
				addPackage(&v4.Package{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addPackage(&v4.Package{Id: "bar", Cpe: "something weird"}),
				addPackageDEPRECATED(&v4.Package{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addPackageDEPRECATED(&v4.Package{Id: "bar", Cpe: "something weird"}),
			},
		},
		"when a package does not have a source package": {
			argOpts: []opts{
				addPackage(&v4.Package{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addPackageDEPRECATED(&v4.Package{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
			},
		},
		"when a package has a source package with another source package": {
			wantErr: `Contents.Packages element "foo": package ID="foo" has a source with a source`,
			argOpts: []opts{
				addPackage(&v4.Package{
					Id:     "foo",
					Cpe:    "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					Source: &v4.Package{Id: "foobar", Source: &v4.Package{}},
				}),
				addPackageDEPRECATED(&v4.Package{
					Id:     "foo",
					Cpe:    "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					Source: &v4.Package{Id: "foobar", Source: &v4.Package{}},
				}),
			},
		},
		"when one of the distributions is empty": {
			wantErr: "Contents.Distributions element \"\" is empty",
			argOpts: []opts{
				addDistribution(&v4.Distribution{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addDistribution(nil),
				addDistributionDEPRECATED(&v4.Distribution{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addDistributionDEPRECATED(nil),
			},
		},
		"when one of the distributions doesn't have id": {
			wantErr: "Contents.Distributions element \"\": ID is empty",
			argOpts: []opts{
				addDistribution(&v4.Distribution{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addDistribution(&v4.Distribution{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addDistributionDEPRECATED(&v4.Distribution{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addDistributionDEPRECATED(&v4.Distribution{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
			},
		},
		"when one of the distributions has an invalid CPE": {
			wantErr: `Contents.Distributions element "bar": invalid CPE: cpe: string does not appear to be a bound WFN: "something weird"`,
			argOpts: []opts{
				addDistribution(&v4.Distribution{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addDistribution(&v4.Distribution{Id: "bar", Cpe: "something weird"}),
				addDistributionDEPRECATED(&v4.Distribution{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addDistributionDEPRECATED(&v4.Distribution{Id: "bar", Cpe: "something weird"}),
			},
		},
		"when one of the repositories is empty": {
			wantErr: "Contents.Repositories element \"\" is empty",
			argOpts: []opts{
				addRepository(&v4.Repository{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addRepository(nil),
				addRepositoryDEPRECATED(&v4.Repository{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addRepositoryDEPRECATED(nil),
			},
		},
		"when one of the repositories doesn't have an id": {
			wantErr: "Contents.Repositories element \"\": ID is empty",
			argOpts: []opts{
				addRepository(&v4.Repository{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addRepository(&v4.Repository{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addRepositoryDEPRECATED(&v4.Repository{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addRepositoryDEPRECATED(&v4.Repository{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
			},
		},
		"when one of the repositories has an invalid CPE": {
			wantErr: `Contents.Repositories element "bar": invalid CPE: cpe: string does not appear to be a bound WFN: "something weird"`,
			argOpts: []opts{
				addRepository(&v4.Repository{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addRepository(&v4.Repository{Id: "bar", Cpe: "something weird"}),
				addRepositoryDEPRECATED(&v4.Repository{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addRepositoryDEPRECATED(&v4.Repository{Id: "bar", Cpe: "something weird"}),
			},
		},
		"when one of the envs reference an invalid valid layer digest": {
			argOpts: []opts{
				addEnvironment("foo", &v4.Environment{
					IntroducedIn: "sha256:0f2e5032c45d68a9585f035970708802bbc7e2688e1552a4c3d8a7c38c3090c3",
				}),
				addEnvironment("bar", &v4.Environment{
					IntroducedIn: "sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e",
				}),
				addEnvironmentDEPRECATED("foo", &v4.Environment{
					IntroducedIn: "sha256:0f2e5032c45d68a9585f035970708802bbc7e2688e1552a4c3d8a7c38c3090c3",
				}),
				addEnvironmentDEPRECATED("bar", &v4.Environment{
					IntroducedIn: "sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e",
				}),
			},
		},
		"when all the envs reference valid layer digests": {
			argOpts: []opts{
				addEnvironment("foo", &v4.Environment{
					IntroducedIn: "sha256:0f2e5032c45d68a9585f035970708802bbc7e2688e1552a4c3d8a7c38c3090c3",
				}),
				addEnvironment("bar", &v4.Environment{
					IntroducedIn: "unexpected layer digest",
				}),
				addEnvironmentDEPRECATED("foo", &v4.Environment{
					IntroducedIn: "sha256:0f2e5032c45d68a9585f035970708802bbc7e2688e1552a4c3d8a7c38c3090c3",
				}),
				addEnvironmentDEPRECATED("bar", &v4.Environment{
					IntroducedIn: "unexpected layer digest",
				}),
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// If not set but with options, use default.
			if tt.arg == nil && len(tt.argOpts) > 0 {
				tt.arg = &v4.GetVulnerabilitiesRequest{
					HashId: "/v4/containerimage/foobar",
					Contents: &v4.Contents{
						Packages:      map[string]*v4.Package{},
						Distributions: map[string]*v4.Distribution{},
						Repositories:  map[string]*v4.Repository{},
						Environments:  map[string]*v4.Environment_List{},
					},
				}
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
			req:     &v4.GetSBOMRequest{Id: "id"},
			wantErr: "name is required",
		},
		"error on no uri": {
			req:     &v4.GetSBOMRequest{Id: "id", Name: "name"},
			wantErr: "uri is required",
		},
		"error on empty contentx": {
			req: &v4.GetSBOMRequest{Id: "id", Name: "name", Uri: "uri"},

			wantErr: "contents are required",
		},
		// This test ensures that the validation logic is executed on the request contents.
		// We do not exercise every possible path where contents are invalid since
		// those paths are already tested as part of Test_validateGetVulnerabilitiesRequest.
		"error on invalid contents": {
			req: &v4.GetSBOMRequest{
				Id:   "id",
				Name: "name",
				Uri:  "uri",
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"": {},
					},
				},
			},
			wantErr: "Contents.Packages element \"\": ID is empty",
		},
		"no error on valid req": {
			req: &v4.GetSBOMRequest{
				Id:       "id",
				Name:     "name",
				Uri:      "uri",
				Contents: &v4.Contents{},
			},
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
