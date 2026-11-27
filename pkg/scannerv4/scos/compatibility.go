package scos

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/quay/claircore/toolkit/types/cpe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
)

// AddCompatibility prepares a copy of a SCOS RPM index for RHEL advisory matching.
func AddCompatibility(report *v4.IndexReport, profile Profile, mapping *repositorytocpe.MappingFile) (*v4.IndexReport, error) {
	if report.GetContents() == nil {
		return nil, fmt.Errorf("index report has no contents")
	}
	report = report.CloneVT()
	contents := report.GetContents()
	dist := &v4.Distribution{
		Id:        DistributionID,
		Did:       DistributionID,
		Name:      Name,
		VersionId: profile.Major,
		Arch:      profile.Arch,
		Cpe:       "cpe:/o:centos:centos:" + profile.Major,
	}
	repositories := make(map[string]*v4.Repository)
	var repositoryIDs []string
	for _, channel := range []string{"baseos", "appstream"} {
		repoID := fmt.Sprintf("rhel-%s-for-%s-%s-rpms", profile.Major, profile.Arch, channel)
		cpes, ok := mapping.GetCPEs(repoID)
		if !ok || len(cpes) == 0 {
			return nil, fmt.Errorf("repository-to-CPE mapping is missing %q", repoID)
		}
		for _, rawCPE := range cpes {
			if _, err := cpe.Unbind(rawCPE); err != nil {
				return nil, fmt.Errorf("parsing CPE for %q: %w", repoID, err)
			}
			id := fmt.Sprintf("%s:%d", DistributionID, len(repositoryIDs))
			repositories[id] = &v4.Repository{
				Id:   id,
				Name: rawCPE,
				Key:  "rhel-cpe-repository",
				Uri:  url.Values{"repoid": {repoID}}.Encode(),
				Cpe:  rawCPE,
			}
			repositoryIDs = append(repositoryIDs, id)
		}
	}
	contents.Distributions = map[string]*v4.Distribution{dist.GetId(): dist}
	contents.DistributionsDEPRECATED = []*v4.Distribution{dist}
	contents.Repositories = repositories
	contents.RepositoriesDEPRECATED = make([]*v4.Repository, 0, len(repositoryIDs))
	for _, id := range repositoryIDs {
		contents.RepositoriesDEPRECATED = append(contents.RepositoriesDEPRECATED, repositories[id])
	}
	environments := contents.GetEnvironments()
	if len(environments) == 0 {
		environments = contents.GetEnvironmentsDEPRECATED()
	}
	var hasRPMs bool
	for _, list := range environments {
		for _, env := range list.GetEnvironments() {
			env.RepositoryIds = nil
			env.DistributionId = dist.GetId()
			if strings.HasPrefix(env.GetPackageDb(), "sqlite:") || strings.HasPrefix(env.GetPackageDb(), "bdb:") {
				env.RepositoryIds = repositoryIDs
				hasRPMs = true
			}
		}
	}
	if !hasRPMs {
		return nil, fmt.Errorf("index report has no RPM environments")
	}
	contents.Environments = environments
	contents.EnvironmentsDEPRECATED = environments
	return report, nil
}
