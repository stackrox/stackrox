package vm

import (
	"github.com/quay/claircore"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

// toProtoV4IndexReport converts claircore.IndexReport to v4.IndexReport.
// This is a minimal conversion for VM scanning without heavy dependencies.
func toProtoV4IndexReport(r *claircore.IndexReport) *v4.IndexReport {
	if r == nil {
		return nil
	}

	contents := toProtoV4Contents(r.Packages, r.Distributions, r.Repositories, r.Environments)
	return &v4.IndexReport{
		State:    r.State,
		Success:  r.Success,
		Err:      r.Err,
		Contents: contents,
	}
}

func toProtoV4Contents(
	ccPkgs map[string]*claircore.Package,
	ccDists map[string]*claircore.Distribution,
	ccRepos map[string]*claircore.Repository,
	ccEnvs map[string][]*claircore.Environment,
) *v4.Contents {
	packages, deprecatedPackages := toV4Packages(ccPkgs)
	distributions, deprecatedDistributions := toV4Distributions(ccDists)
	repositories, deprecatedRepositories := toV4Repositories(ccRepos)
	environments, deprecatedEnvironments := toV4Environments(ccEnvs)

	return &v4.Contents{
		Packages:                packages,
		PackagesDEPRECATED:      deprecatedPackages,
		Distributions:           distributions,
		DistributionsDEPRECATED: deprecatedDistributions,
		Repositories:            repositories,
		RepositoriesDEPRECATED:  deprecatedRepositories,
		Environments:            environments,
		EnvironmentsDEPRECATED:  deprecatedEnvironments,
	}
}

func toV4Packages(ccPkgs map[string]*claircore.Package) (map[string]*v4.Package, []*v4.Package) {
	if len(ccPkgs) == 0 {
		return nil, nil
	}

	packages := make(map[string]*v4.Package, len(ccPkgs))
	deprecatedPackages := make([]*v4.Package, 0, len(ccPkgs))

	for id, ccPkg := range ccPkgs {
		v4Pkg := toV4Package(ccPkg)
		packages[id] = v4Pkg
		deprecatedPackages = append(deprecatedPackages, v4Pkg)
	}
	return packages, deprecatedPackages
}

func toV4Package(p *claircore.Package) *v4.Package {
	if p == nil {
		return nil
	}

	var srcPkg *v4.Package
	if p.Source != nil {
		srcPkg = toV4Package(p.Source)
	}

	var normalizedVersion *v4.NormalizedVersion
	if len(p.NormalizedVersion.V) > 0 {
		normalizedVersion = &v4.NormalizedVersion{
			Kind: p.NormalizedVersion.Kind,
			V:    p.NormalizedVersion.V[:],
		}
	}

	cpeString := p.CPE.BindFS()

	return &v4.Package{
		Id:                p.ID,
		Name:              p.Name,
		Version:           p.Version,
		NormalizedVersion: normalizedVersion,
		Kind:              p.Kind,
		Source:            srcPkg,
		PackageDb:         p.PackageDB,
		RepositoryHint:    p.RepositoryHint,
		Module:            p.Module,
		Arch:              p.Arch,
		Cpe:               cpeString,
	}
}

func toV4Distributions(ccDists map[string]*claircore.Distribution) (map[string]*v4.Distribution, []*v4.Distribution) {
	if len(ccDists) == 0 {
		return nil, nil
	}

	distributions := make(map[string]*v4.Distribution, len(ccDists))
	deprecatedDistributions := make([]*v4.Distribution, 0, len(ccDists))

	for id, ccDist := range ccDists {
		v4Dist := toV4Distribution(ccDist)
		distributions[id] = v4Dist
		deprecatedDistributions = append(deprecatedDistributions, v4Dist)
	}
	return distributions, deprecatedDistributions
}

func toV4Distribution(d *claircore.Distribution) *v4.Distribution {
	if d == nil {
		return nil
	}

	// Extract version ID with Alpine fallback
	vID := d.VersionID
	if vID == "" && d.DID == "alpine" {
		vID = d.Version
	}

	cpeString := d.CPE.BindFS()

	return &v4.Distribution{
		Id:              d.ID,
		Did:             d.DID,
		Name:            d.Name,
		Version:         d.Version,
		VersionCodeName: d.VersionCodeName,
		VersionId:       vID,
		Arch:            d.Arch,
		Cpe:             cpeString,
		PrettyName:      d.PrettyName,
	}
}

func toV4Repositories(ccRepos map[string]*claircore.Repository) (map[string]*v4.Repository, []*v4.Repository) {
	if len(ccRepos) == 0 {
		return nil, nil
	}

	repositories := make(map[string]*v4.Repository, len(ccRepos))
	deprecatedRepositories := make([]*v4.Repository, 0, len(ccRepos))

	for id, ccRepo := range ccRepos {
		v4Repo := toV4Repository(ccRepo)
		repositories[id] = v4Repo
		deprecatedRepositories = append(deprecatedRepositories, v4Repo)
	}
	return repositories, deprecatedRepositories
}

func toV4Repository(r *claircore.Repository) *v4.Repository {
	if r == nil {
		return nil
	}

	cpeString := r.CPE.BindFS()

	return &v4.Repository{
		Id:   r.ID,
		Name: r.Name,
		Key:  r.Key,
		Uri:  r.URI,
		Cpe:  cpeString,
	}
}

func toV4Environments(ccEnvs map[string][]*claircore.Environment) (map[string]*v4.Environment_List, map[string]*v4.Environment_List) {
	if len(ccEnvs) == 0 {
		return nil, nil
	}

	environments := make(map[string]*v4.Environment_List)
	deprecatedEnvironments := make(map[string]*v4.Environment_List)

	for pkgID, envs := range ccEnvs {
		if len(envs) == 0 {
			continue
		}

		v4Envs := make([]*v4.Environment, 0, len(envs))
		for _, env := range envs {
			v4Env := &v4.Environment{
				PackageDb:      env.PackageDB,
				IntroducedIn:   toV4Digest(env.IntroducedIn),
				DistributionId: env.DistributionID,
				RepositoryIds:  env.RepositoryIDs,
			}
			v4Envs = append(v4Envs, v4Env)
		}

		envList := &v4.Environment_List{Environments: v4Envs}
		environments[pkgID] = envList
		deprecatedEnvironments[pkgID] = envList
	}

	return environments, deprecatedEnvironments
}

func toV4Digest(d claircore.Digest) string {
	if d.String() == "" {
		return ""
	}
	return d.String()
}
