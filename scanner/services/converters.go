package services

import (
	"github.com/quay/claircore"
	"github.com/quay/claircore/pkg/cpe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

func convertToIndexReport(r *claircore.IndexReport) *v4.IndexReport {
	if r == nil {
		return nil
	}
	var environments map[string]*v4.Environment_List
	if len(r.Environments) > 0 {
		environments = make(map[string]*v4.Environment_List, len(r.Environments))
	}
	for k, v := range r.Environments {
		for _, e := range v {
			l, ok := environments[k]
			if !ok {
				l = &v4.Environment_List{}
				environments[k] = l
			}
			l.Environments = append(l.Environments, convertToEnvironment(e))
		}
	}
	return &v4.IndexReport{
		State:   r.State,
		Success: r.Success,
		Err:     r.Err,
		Contents: &v4.Contents{
			Packages:      convertMapToSlice(convertToPackage, r.Packages),
			Distributions: convertMapToSlice(convertToDistribution, r.Distributions),
			Repositories:  convertMapToSlice(convertToRepository, r.Repositories),
			Environments:  environments,
		},
	}
}

func convertToPackage(p *claircore.Package) *v4.Package {
	if p == nil {
		return nil
	}
	// Conversion functions.
	toNormalizedVersion := func(version claircore.Version) *v4.NormalizedVersion {
		return &v4.NormalizedVersion{
			Kind: version.Kind,
			V:    version.V[:],
		}
	}
	toSourcePackage := func(p *claircore.Package) (source *v4.Package) {
		if p == nil {
			return nil
		}
		// Sanitize and avoid recursion.
		if p.Source != nil {
			p.Source.Source = nil
			source = convertToPackage(p.Source)
		}
		return source
	}
	return &v4.Package{
		Id:                p.ID,
		Name:              p.Name,
		Version:           p.Version,
		NormalizedVersion: toNormalizedVersion(p.NormalizedVersion),
		Kind:              p.Kind,
		Source:            toSourcePackage(p.Source),
		PackageDb:         p.PackageDB,
		RepositoryHint:    p.RepositoryHint,
		Module:            p.Module,
		Arch:              p.Arch,
		Cpe:               convertToCPEString(p.CPE),
	}
}

func convertToDistribution(d *claircore.Distribution) *v4.Distribution {
	if d == nil {
		return nil
	}
	return &v4.Distribution{
		Id:              d.ID,
		Did:             d.DID,
		Name:            d.Name,
		Version:         d.Version,
		VersionCodeName: d.VersionCodeName,
		VersionId:       d.VersionID,
		Arch:            d.Arch,
		Cpe:             convertToCPEString(d.CPE),
		PrettyName:      d.PrettyName,
	}
}

func convertToRepository(r *claircore.Repository) *v4.Repository {
	if r == nil {
		return nil
	}
	return &v4.Repository{
		Id:   r.ID,
		Name: r.Name,
		Key:  r.Key,
		Uri:  r.URI,
		Cpe:  convertToCPEString(r.CPE),
	}
}

func convertToEnvironment(e *claircore.Environment) *v4.Environment {
	if e == nil {
		return nil
	}
	return &v4.Environment{
		PackageDb:      e.PackageDB,
		IntroducedIn:   convertToDigestString(e.IntroducedIn),
		DistributionId: e.DistributionID,
		RepositoryIds:  append([]string(nil), e.RepositoryIDs...),
	}
}

func convertToCPEString(c cpe.WFN) string {
	return c.String()
}

func convertToDigestString(digest claircore.Digest) string {
	return digest.String()
}

// convertMapToSlice converts generic maps keyed by strings to a slice using a
// provided conversion function.
func convertMapToSlice[IN any, OUT any](convF func(*IN) *OUT, in map[string]*IN) (out []*OUT) {
	for _, i := range in {
		out = append(out, convF(i))
	}
	return out
}
