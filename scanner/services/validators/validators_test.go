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
			r.Contents.Packages = append(r.Contents.Packages, p)
		})
	}
	addDistribution := func(d *v4.Distribution) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.Contents.Distributions = append(r.Contents.Distributions, d)
		})
	}
	addRepository := func(d *v4.Repository) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			r.Contents.Repositories = append(r.Contents.Repositories, d)
		})
	}
	addEnvironment := func(pkgId string, d *v4.Environment) opts {
		return withContent(func(r *v4.GetVulnerabilitiesRequest) {
			if r.Contents.Environments == nil {
				r.Contents.Environments = make(map[string]*v4.Environment_List)
			}
			envs, ok := r.Contents.Environments[pkgId]
			if !ok {
				envs = &v4.Environment_List{}
				r.Contents.Environments[pkgId] = envs
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
			wantErr: "Contents.Packages element #2 is empty",
			argOpts: []opts{
				addPackage(&v4.Package{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addPackage(nil),
			},
		},
		"when one of the packages doesn't have id": {
			wantErr: `Contents.Packages element #2: Id is empty`,
			argOpts: []opts{
				addPackage(&v4.Package{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addPackage(&v4.Package{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
			},
		},
		"when one of the packages has an invalid CPE": {
			wantErr: `Contents.Packages element #2 (id: "bar"): invalid CPE`,
			argOpts: []opts{
				addPackage(&v4.Package{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addPackage(&v4.Package{Id: "bar", Cpe: "something weird"}),
			},
		},
		"when a package does not have a source package": {
			argOpts: []opts{
				addPackage(&v4.Package{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
			},
		},
		"when a package has a source package with another source package": {
			wantErr: `Contents.Packages element #1 (id: "foo"): package ID="foo" has a source with a source`,
			argOpts: []opts{
				addPackage(&v4.Package{
					Id:     "foo",
					Cpe:    "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					Source: &v4.Package{Id: "foobar", Source: &v4.Package{}},
				}),
			},
		},
		"when one of the distributions is empty": {
			wantErr: "Contents.Distributions element #2 is empty",
			argOpts: []opts{
				addDistribution(&v4.Distribution{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addDistribution(nil),
			},
		},
		"when one of the distributions doesn't have id": {
			wantErr: "Contents.Distributions element #2: Id is empty",
			argOpts: []opts{
				addDistribution(&v4.Distribution{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addDistribution(&v4.Distribution{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
			},
		},
		"when one of the distributions has an invalid CPE": {
			wantErr: `Contents.Distributions element #2 (id: "bar"): invalid CPE`,
			argOpts: []opts{
				addDistribution(&v4.Distribution{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addDistribution(&v4.Distribution{Id: "bar", Cpe: "something weird"}),
			},
		},
		"when one of the repositories is empty": {
			wantErr: "Contents.Repositories element #2 is empty",
			argOpts: []opts{
				addRepository(&v4.Repository{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addRepository(nil),
			},
		},
		"when one of the repositories doesn't have an id": {
			wantErr: "Contents.Repositories element #2: Id is empty",
			argOpts: []opts{
				addRepository(&v4.Repository{Id: "foobar", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addRepository(&v4.Repository{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
			},
		},
		"when one of the repositories has an invalid CPE": {
			wantErr: `Contents.Repositories element #2 (id: "bar"): invalid CPE`,
			argOpts: []opts{
				addRepository(&v4.Repository{Id: "foo", Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"}),
				addRepository(&v4.Repository{Id: "bar", Cpe: "something weird"}),
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
						Packages:      []*v4.Package{},
						Distributions: []*v4.Distribution{},
						Repositories:  []*v4.Repository{},
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
