package mappers

import (
	"github.com/stackrox/rox/clair-adapter/clairclient"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

// ToProtoIndexReport converts a Clair IndexReport to the Scanner V4 proto format.
func ToProtoIndexReport(ir *clairclient.IndexReport) (*v4.IndexReport, error) {
	contents, err := toProtoContents(
		ir.Packages,
		ir.Distributions,
		ir.Repositories,
		ir.Environments,
		nil, // pkgFixedBy is nil for index reports
	)
	if err != nil {
		return nil, err
	}

	return &v4.IndexReport{
		HashId:   ir.ManifestHash,
		State:    ir.State,
		Success:  ir.Success,
		Err:      ir.Err,
		Contents: contents,
	}, nil
}

// toProtoContents converts Clair content maps to proto Contents.
// This is shared between index and vulnerability report mappers.
// pkgFixedBy maps package ID to fixed-in version (nil for index reports).
func toProtoContents(
	packages map[string]clairclient.Package,
	distributions map[string]clairclient.Distribution,
	repositories map[string]clairclient.Repository,
	environments map[string][]clairclient.Environment,
	pkgFixedBy map[string]string,
) (*v4.Contents, error) {
	contents := &v4.Contents{
		Packages:      make(map[string]*v4.Package),
		Distributions: make(map[string]*v4.Distribution),
		Repositories:  make(map[string]*v4.Repository),
		Environments:  make(map[string]*v4.Environment_List),
	}

	// Convert packages
	for id, pkg := range packages {
		var fixedIn string
		if pkgFixedBy != nil {
			fixedIn = pkgFixedBy[id]
		}
		contents.Packages[id] = toProtoPackage(pkg, fixedIn)
	}

	// Convert distributions
	for id, dist := range distributions {
		contents.Distributions[id] = toProtoDistribution(dist)
	}

	// Convert repositories
	for id, repo := range repositories {
		contents.Repositories[id] = toProtoRepository(repo)
	}

	// Convert environments
	for id, envList := range environments {
		protoEnvList := &v4.Environment_List{
			Environments: make([]*v4.Environment, 0, len(envList)),
		}
		for _, env := range envList {
			protoEnvList.Environments = append(protoEnvList.Environments, toProtoEnvironment(env))
		}
		contents.Environments[id] = protoEnvList
	}

	return contents, nil
}

// toProtoPackage converts a Clair Package to proto Package.
// Handles recursive Source field and int to int32 conversion for NormalizedVersion.
func toProtoPackage(pkg clairclient.Package, fixedInVersion string) *v4.Package {
	protoPkg := &v4.Package{
		Id:             pkg.ID,
		Name:           pkg.Name,
		Version:        pkg.Version,
		Kind:           pkg.Kind,
		Arch:           pkg.Arch,
		Module:         pkg.Module,
		Cpe:            pkg.CPE,
		PackageDb:      pkg.PackageDB,
		RepositoryHint: pkg.RepositoryHint,
		FixedInVersion: fixedInVersion,
	}

	// Handle recursive source package
	if pkg.Source != nil {
		protoPkg.Source = toProtoPackage(*pkg.Source, "")
	}

	// Convert NormalizedVersion (int -> int32)
	if pkg.NormalizedVersion.Kind != "" || len(pkg.NormalizedVersion.V) > 0 {
		protoPkg.NormalizedVersion = &v4.NormalizedVersion{
			Kind: pkg.NormalizedVersion.Kind,
			V:    make([]int32, len(pkg.NormalizedVersion.V)),
		}
		for i, v := range pkg.NormalizedVersion.V {
			protoPkg.NormalizedVersion.V[i] = int32(v)
		}
	}

	return protoPkg
}

// toProtoDistribution converts a Clair Distribution to proto Distribution.
func toProtoDistribution(dist clairclient.Distribution) *v4.Distribution {
	return &v4.Distribution{
		Id:              dist.ID,
		Did:             dist.DID,
		Name:            dist.Name,
		Version:         dist.Version,
		VersionCodeName: dist.VersionCodeName,
		VersionId:       dist.VersionID,
		Arch:            dist.Arch,
		Cpe:             dist.CPE,
		PrettyName:      dist.PrettyName,
	}
}

// toProtoRepository converts a Clair Repository to proto Repository.
func toProtoRepository(repo clairclient.Repository) *v4.Repository {
	return &v4.Repository{
		Id:   repo.ID,
		Name: repo.Name,
		Key:  repo.Key,
		Uri:  repo.URI,
		Cpe:  repo.CPE,
	}
}

// toProtoEnvironment converts a Clair Environment to proto Environment.
func toProtoEnvironment(env clairclient.Environment) *v4.Environment {
	return &v4.Environment{
		PackageDb:      env.PackageDB,
		IntroducedIn:   env.IntroducedIn,
		DistributionId: env.DistributionID,
		RepositoryIds:  env.RepositoryIDs,
	}
}
