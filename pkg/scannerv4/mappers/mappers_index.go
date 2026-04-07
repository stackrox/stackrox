package mappers

import (
	"fmt"

	"github.com/quay/claircore"
	"github.com/quay/claircore/toolkit/types/cpe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

const (
	rhelRepositoryKey = "rhel-cpe-repository"
)

// ToProtoV4IndexReport maps claircore.IndexReport to v4.IndexReport.
func ToProtoV4IndexReport(r *claircore.IndexReport) (*v4.IndexReport, error) {
	if r == nil {
		return nil, nil
	}
	contents, err := toProtoV4IndexContents(r.Packages, r.Distributions, r.Repositories, r.Environments)
	if err != nil {
		return nil, err
	}
	return &v4.IndexReport{
		State:    r.State,
		Success:  r.Success,
		Err:      r.Err,
		Contents: contents,
	}, nil
}

// toProtoV4IndexContents converts index report contents without enrichment data.
// This is a lightweight version used by roxagent that doesn't require pkgFixedBy enrichment.
func toProtoV4IndexContents(
	pkgs map[string]*claircore.Package,
	dists map[string]*claircore.Distribution,
	repos map[string]*claircore.Repository,
	envs map[string][]*claircore.Environment,
) (*v4.Contents, error) {
	packages, deprecatedPackages, err := v4IndexPackages(pkgs)
	if err != nil {
		return nil, err
	}
	distributions, deprecatedDistributions, err := claircoreToV4(dists, v4Distribution)
	if err != nil {
		return nil, err
	}
	repositories, deprecatedRepositories, err := claircoreToV4(repos, v4Repository)
	if err != nil {
		return nil, err
	}
	environments, deprecatedEnrivonments := v4Environments(envs, repos)
	return &v4.Contents{
		Packages:                packages,
		PackagesDEPRECATED:      deprecatedPackages,
		Distributions:           distributions,
		DistributionsDEPRECATED: deprecatedDistributions,
		Repositories:            repositories,
		RepositoriesDEPRECATED:  deprecatedRepositories,
		Environments:            environments,
		EnvironmentsDEPRECATED:  deprecatedEnrivonments,
	}, nil
}

func v4IndexPackages(ccPkgs map[string]*claircore.Package) (map[string]*v4.Package, []*v4.Package, error) {
	if len(ccPkgs) == 0 {
		return nil, nil, nil
	}
	packages := make(map[string]*v4.Package, len(ccPkgs))
	deprecatedPackages := make([]*v4.Package, 0, len(ccPkgs))
	for id, ccPkg := range ccPkgs {
		v4Pkg, err := v4Package(ccPkg)
		if err != nil {
			return nil, nil, err
		}
		packages[id] = v4Pkg
		deprecatedPackages = append(deprecatedPackages, v4Pkg)
	}
	return packages, deprecatedPackages, nil
}

func v4Package(p *claircore.Package) (*v4.Package, error) {
	if p == nil {
		return nil, nil
	}
	if p.Source != nil && p.Source.Source != nil {
		return nil, fmt.Errorf("package %q: invalid source package %q: source specifies source",
			p.ID, p.Source.ID)
	}
	// Conversion function.
	toNormalizedVersion := func(version claircore.Version) *v4.NormalizedVersion {
		return &v4.NormalizedVersion{
			Kind: version.Kind,
			V:    version.V[:],
		}
	}
	srcPkg, err := v4Package(p.Source)
	if err != nil {
		return nil, err
	}
	return &v4.Package{
		Id:                p.ID,
		Name:              p.Name,
		Version:           p.Version,
		NormalizedVersion: toNormalizedVersion(p.NormalizedVersion),
		Kind:              p.Kind,
		Source:            srcPkg,
		PackageDb:         p.PackageDB,
		RepositoryHint:    p.RepositoryHint,
		Module:            p.Module,
		Arch:              p.Arch,
		Cpe:               toCPEString(p.CPE),
	}, nil
}

// VersionID returns the distribution version ID.
func VersionID(d *claircore.Distribution) string {
	vID := d.VersionID
	if vID == "" {
		switch d.DID {
		// TODO(ROX-21678): `VersionId` is currently not populated for Alpine[1],
		//                  temporarily falling back to the version.
		//
		// [1]: https://github.com/quay/claircore/blob/88ccfbecee88d7b326b9a2fb3ab5b5f4cfa0b610/alpine/distributionscanner.go#L110-L113
		case "alpine":
			vID = d.Version
		}
	}
	return vID
}

func claircoreToV4[K comparable, V1, V2 any](cc map[K]V1, f func(V1) (V2, error)) (map[K]V2, []V2, error) {
	if len(cc) == 0 {
		return nil, nil, nil
	}
	v4Map := make(map[K]V2, len(cc))
	v4Slice := make([]V2, 0, len(cc))
	for k, v := range cc {
		v4Resource, err := f(v)
		if err != nil {
			return nil, nil, err
		}
		v4Map[k] = v4Resource
		v4Slice = append(v4Slice, v4Resource)
	}
	return v4Map, v4Slice, nil
}

func v4Distribution(d *claircore.Distribution) (*v4.Distribution, error) {
	if d == nil {
		return nil, nil
	}
	return &v4.Distribution{
		Id:              d.ID,
		Did:             d.DID,
		Name:            d.Name,
		Version:         d.Version,
		VersionCodeName: d.VersionCodeName,
		VersionId:       VersionID(d),
		Arch:            d.Arch,
		Cpe:             toCPEString(d.CPE),
		PrettyName:      d.PrettyName,
	}, nil
}

func v4Repository(r *claircore.Repository) (*v4.Repository, error) {
	if r == nil {
		return nil, nil
	}
	return &v4.Repository{
		Id:   r.ID,
		Name: r.Name,
		Key:  r.Key,
		Uri:  r.URI,
		Cpe:  toCPEString(r.CPE),
	}, nil
}

func v4Environments(ccEnvs map[string][]*claircore.Environment, ccRepos map[string]*claircore.Repository) (map[string]*v4.Environment_List, map[string]*v4.Environment_List) {
	if len(ccEnvs) == 0 {
		return nil, nil
	}
	environments := make(map[string]*v4.Environment_List, len(ccEnvs))
	environmentsDeprecated := make(map[string]*v4.Environment_List, len(ccEnvs))
	for id, envs := range ccEnvs {
		l, ok := environments[id]
		lDeprecated := environmentsDeprecated[id]
		if !ok {
			l = &v4.Environment_List{}
			environments[id] = l
			lDeprecated = &v4.Environment_List{}
			environmentsDeprecated[id] = lDeprecated
		}
		for _, env := range envs {
			l.Environments = append(l.Environments, v4Environment(env))
			lDeprecated.Environments = append(lDeprecated.Environments, v4EnvironmentDeprecated(env, ccRepos))
		}
	}
	return environments, environmentsDeprecated
}

func v4Environment(e *claircore.Environment) *v4.Environment {
	if e == nil {
		return nil
	}
	return &v4.Environment{
		PackageDb:      e.PackageDB,
		IntroducedIn:   toDigestString(e.IntroducedIn),
		DistributionId: e.DistributionID,
		RepositoryIds:  append([]string(nil), e.RepositoryIDs...),
	}
}

func v4EnvironmentDeprecated(e *claircore.Environment, repos map[string]*claircore.Repository) *v4.Environment {
	if e == nil {
		return nil
	}
	repoIDs := make([]string, 0, len(e.RepositoryIDs))
	for _, id := range e.RepositoryIDs {
		repo, ok := repos[id]
		if !ok {
			continue
		}
		// In Claircore v1.5.40+, the repositories are no longer all keyed by ID.
		// RPMs in RHEL-based containers are now keyed by name.
		// In older ACS versions, we assumed all repos were keyed by ID,
		// so if the key is not the ID, we check if it's actually the name
		// and this is, in fact, a RHEL RPM.
		if repo.Key == rhelRepositoryKey {
			repoIDs = append(repoIDs, repo.ID)
			continue
		}
		repoIDs = append(repoIDs, id)
	}
	return &v4.Environment{
		PackageDb:      e.PackageDB,
		IntroducedIn:   toDigestString(e.IntroducedIn),
		DistributionId: e.DistributionID,
		RepositoryIds:  repoIDs,
	}
}

func toCPEString(c cpe.WFN) string {
	return c.BindFS()
}

func toDigestString(digest claircore.Digest) string {
	return digest.String()
}
