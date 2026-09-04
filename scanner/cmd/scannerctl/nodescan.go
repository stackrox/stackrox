package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

func nodeScanCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "nodescan",
		Short: "Triggers a node scan for the node the target node indexer runs on",
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Create scanner client.
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}
		if len(args) != 1 {
			return errors.New("expected two arguments: rhcos_version")
		}

		var ir = createIndexReport(args[0])
		nodeDigest, err := name.NewDigest("registry.redhat.io/rhcos@" + ir.GetHashId())
		if err != nil {
			return fmt.Errorf("failed to parse const node digest %w", err)
		}
		_ = nodeDigest
		digest := nodeDigest

		vr, err := scanner.GetVulnerabilities(ctx, digest, ir.GetContents())
		if err != nil {
			return fmt.Errorf("scanning: %w", err)
		}
		reportJSON, err := json.MarshalIndent(vr, "", "  ")
		if err != nil {
			return fmt.Errorf("decoding report: %w", err)
		}
		_ = reportJSON
		// fmt.Println(string(reportJSON))
		fmt.Printf("Vulnerability Report contains %d packages with %d vulns and %d package vulns\n",
			len(vr.GetContents().GetPackages()),
			len(vr.GetVulnerabilities()),
			len(vr.GetPackageVulnerabilities()))

		fmt.Printf("%s\n", vr.GetPackageVulnerabilities())

		for pkgID, vulnList := range vr.GetPackageVulnerabilities() {
			pkgs := vr.GetContents().GetPackages()
			pkg := &v4.Package{}
			for _, p := range pkgs {
				if p.GetId() == pkgID {
					pkg = p
				}
			}
			fmt.Printf("pkg: (%s)%s [v%s]\n", pkg.GetId(), pkg.GetName(), pkg.GetVersion())
			for i, vulnID := range vulnList.GetValues() {
				for vID, vuln := range vr.GetVulnerabilities() {
					if vID == vulnID {
						fmt.Printf("\t(%d) Vuln: %s\n", i, vuln.GetName())
					}
				}
			}
		}
		return nil
	}
	return &cmd
}

func normalizeVersion(version string) []int32 {
	fields := strings.Split(version, ".")
	switch len(fields) {
	case 0:
		return []int32{0, 0, 0}
	case 1:
		i, err := strconv.ParseInt(fields[0], 10, 32)
		if err == nil {
			return []int32{int32(i), 0, 0}
		}
	}
	i1, err1 := strconv.ParseInt(fields[0], 10, 32)
	if err1 != nil {
		i1 = 0
	}
	i2, err2 := strconv.ParseInt(fields[1], 10, 32)
	if err2 != nil {
		i2 = 0
	}
	// Only two first fields matter for the db query.
	// Final decision is done with full string compare.
	return []int32{int32(i1), int32(i2), 0}
}

func createIndexReport(version string) *v4.IndexReport {
	const goldenName = "Red Hat Container Catalog"
	const goldenURI = `https://catalog.redhat.com/software/containers/explore`

	rep := &v4.IndexReport{
		HashId: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		//HashId:  "sha256:11cf2360bc7d8d4fef440b3fa97ce49cd648318632328f42ecbfb071b823ae14",
		State:   "7", // IndexFinished
		Success: true,
		Err:     "",
		Contents: &v4.Contents{
			Packages: []*v4.Package{
				{
					Id:      "66",
					Name:    "expat",
					Version: "2.5.0-2.el9_4.2",
					Kind:    "binary",
					Source: &v4.Package{
						Name:    "expat",
						Version: "2.5.0-2.el9_4.2",
						Kind:    "source",
						Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
					PackageDb:      "sqlite:usr/share/rpm",
					RepositoryHint: "hash:sha256:f78a31433453e6966842a852ebcb9eac39cbd66e76c8062ceb9c71bbf0f52cba|key:199e2f91fd431d51",
					Module:         "",
					Arch:           "x86_64",
					Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				},
				//{
				//	Id:      "43",
				//	Name:    "libstdc++",
				//	Version: "11.4.1-3.el9",
				//	Kind:    "binary",
				//	Source: &v4.Package{
				//		Name:    "gcc",
				//		Version: "11.4.1-3.el9",
				//		Kind:    "source",
				//		Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				//	},
				//	PackageDb:      "sqlite:usr/share/rpm",
				//	RepositoryHint: "hash:sha256:cabc1aaa5dded6ad6775b83847bac394cf13ebc334dd8c676d10a378b70b0db0|key:199e2f91fd431d51",
				//	Module:         "",
				//	Arch:           "x86_64",
				//	Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				//},
			},
			Repositories: []*v4.Repository{
				{
					Id:   "6",
					Name: goldenName,
					Key:  "",
					Uri:  goldenURI,
					Cpe:  "cpe:2.3:*", // required to pass validation
				},
				{
					Id:   "0",
					Name: "cpe:/o:redhat:rhel_eus:9.4::baseos",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:o:redhat:rhel_eus:9.4:*:baseos:*:*:*:*:*",
				},
				{
					Id:   "1",
					Name: "cpe:/a:redhat:rhel_eus:9.4::appstream",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:a:redhat:rhel_eus:9.4:*:appstream:*:*:*:*:*",
				},
				{
					Id:   "2",
					Name: "cpe:/o:redhat:enterprise_linux:9::fastdatapath",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:o:redhat:enterprise_linux:9:*:fastdatapath:*:*:*:*:*",
				},
				{
					Id:   "3",
					Name: "cpe:/a:redhat:openshift:4.17::el9",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:a:redhat:openshift:4.17:*:el9:*:*:*:*:*",
				},
			},
			// Environments must be present for the matcher to discover records
			Environments: map[string]*v4.Environment_List{
				"6": {
					Environments: []*v4.Environment{
						{
							PackageDb:     "",
							IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							RepositoryIds: []string{"6"},
						},
					},
				},
				"43": {
					Environments: []*v4.Environment{
						{
							PackageDb:      "sqlite:usr/share/rpm",
							IntroducedIn:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							DistributionId: "",
							RepositoryIds:  []string{"0", "1", "2"},
						},
					},
				},
				"66": {
					Environments: []*v4.Environment{
						{
							PackageDb:      "sqlite:usr/share/rpm",
							IntroducedIn:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							DistributionId: "",
							RepositoryIds:  []string{"0", "1", "2"},
						},
					},
				},
			},
		},
	}
	if version != "" {
		normVersion := normalizeVersion(version)
		fmt.Fprintf(os.Stderr, "Version: %s, Normalized version: %v\n", version, normVersion)
		rhcosPkg := &v4.Package{
			Id:      "6",
			Name:    "rhcos",
			Version: version, // this will be read!
			NormalizedVersion: &v4.NormalizedVersion{ // both are required
				Kind: "rhctag",
				V:    normVersion,
			},
			FixedInVersion: "",
			Kind:           "binary",
			Source: &v4.Package{
				Id:  "6",
				Cpe: "cpe:2.3:*", // required to pass validation
			},
			Arch: "",
			Cpe:  "cpe:2.3:*", // required to pass validation
		}
		rep.GetContents().Packages = append(rep.GetContents().Packages, rhcosPkg)
	}

	return rep
}
