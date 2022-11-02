///////////////////////////////////////////////////
// Influenced by ClairCore under Apache 2.0 License
// https://github.com/quay/claircore
///////////////////////////////////////////////////

package v1

import (
	"strconv"

	rpmVersion "github.com/knqyf263/go-rpm-version"
	log "github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/scanner/api/v1/common"
	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/ext/featurefmt"
	"github.com/stackrox/scanner/ext/versionfmt/rpm"
	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/types"
)

const (
	timeFormat = "2006-01-02T15:04Z"
)

// addRHELv2Vulns appends vulnerabilities found during RHELv2 scanning.
// RHELv2 scanning performs the scanning/analysis needed to be
// certified as part of Red Hat's Scanner Certification Program.
// The returned bool indicates if full certified scanning was performed.
// This is typically only `false` for images without proper CPE information.
func addRHELv2Vulns(db database.Datastore, layer *Layer) (bool, error) {
	pkgEnvs, cpesExist, err := getRHELv2PkgEnvs(db, layer.Name)
	if err != nil {
		return false, err
	}

	features, err := getFullRHELv2Features(db, pkgEnvs, false)
	if err != nil {
		return false, err
	}

	layer.Features = append(layer.Features, features...)

	return cpesExist, nil
}

func getFullRHELv2Features(db database.Datastore, pkgEnvs map[int]*database.RHELv2PackageEnv, execsPopulated bool) ([]Feature, error) {
	records := getRHELv2Records(pkgEnvs)
	vulns, err := db.GetRHELv2Vulnerabilities(records)
	if err != nil {
		return nil, err
	}

	var features []Feature
	var depMap map[string]common.FeatureKeySet
	if !execsPopulated {
		depMap = common.GetDepMapRHEL(pkgEnvs)
	}
	for _, pkgEnv := range pkgEnvs {
		pkg := pkgEnv.Pkg

		if hasKernelPrefix(pkg.Name) {
			continue
		}

		version := pkg.GetPackageVersion()
		pkgKey := featurefmt.PackageKey{Name: pkg.Name, Version: version}
		executables := pkg.Executables
		if !execsPopulated {
			executables = common.CreateExecutablesFromDependencies(pkgKey, pkg.ExecutableToDependencies, depMap)
		}
		feature := Feature{
			Name:          pkg.Name,
			NamespaceName: pkgEnv.Namespace,
			VersionFormat: rpm.ParserName,
			Version:       version,
			AddedBy:       pkgEnv.AddedBy,
			Executables:   executables,
		}

		feature.FixedBy, feature.Vulnerabilities = getRHELv2Vulns(vulns, pkg, feature.NamespaceName)

		features = append(features, feature)
	}

	return features, nil
}

func getFullFeaturesForRHELv2Packages(db database.Datastore, pkgs []*v1.RHELComponent) ([]Feature, error) {
	pkgEnvs := make(map[int]*database.RHELv2PackageEnv, len(pkgs))
	for _, pkg := range pkgs {
		pkgEnvs[int(pkg.GetId())] = &database.RHELv2PackageEnv{
			Pkg: &database.RHELv2Package{
				Model:       database.Model{ID: int(pkg.GetId())},
				Name:        pkg.GetName(),
				Version:     pkg.GetVersion(),
				Module:      pkg.GetModule(),
				Arch:        pkg.GetArch(),
				Executables: pkg.GetExecutables(),
			},
			Namespace: pkg.GetNamespace(),
			AddedBy:   pkg.GetAddedBy(),
			CPEs:      pkg.GetCpes(),
		}
	}

	return getFullRHELv2Features(db, pkgEnvs, true)
}

// getRHELv2Vulns gets the vulnerabilities and fixedBy version found during RHELv2 scanning.
func getRHELv2Vulns(vulns map[int][]*database.RHELv2Vulnerability, pkg *database.RHELv2Package, namespaceName string) (string, []Vulnerability) {
	pkgVersion := rpmVersion.NewVersion(pkg.Version)
	pkgArch := pkg.Arch
	fixedBy := pkgVersion

	var vulnerabilities []Vulnerability

	// Database query results need more filtering.
	// Need to ensure:
	// 1. The package's version is less than the vuln's fixed-in version, if present.
	// 2. The ArchOperation passes.
	for _, vuln := range vulns[pkg.ID] {
		if len(vuln.PackageInfos) != 1 {
			log.Warnf("Unexpected number of package infos for vuln %q (%d != %d); Skipping...", vuln.Name, len(vuln.PackageInfos), 1)
			continue
		}
		vulnPkgInfo := vuln.PackageInfos[0]

		if len(vulnPkgInfo.Packages) != 1 {
			log.Warnf("Unexpected number of packages for vuln %q (%d != %d); Skipping...", vuln.Name, len(vulnPkgInfo.Packages), 1)
			continue
		}
		vulnPkg := vulnPkgInfo.Packages[0]

		// Assume the vulnerability is not fixed.
		// In that case, all versions are affected.
		affectedVersion := true
		var vulnVersion *rpmVersion.Version
		if vulnPkgInfo.FixedInVersion != "" {
			// The vulnerability is fixed. Determine if this package is affected.
			vulnVersion = rpmVersionPtr(rpmVersion.NewVersion(vulnPkgInfo.FixedInVersion))
			affectedVersion = pkgVersion.LessThan(*vulnVersion)
		}

		// Compare the package's architecture to the affected architecture.
		affectedArch := vulnPkgInfo.ArchOperation.Cmp(pkgArch, vulnPkg.Arch)

		if affectedVersion && affectedArch {
			vulnerabilities = append(vulnerabilities, RHELv2ToVulnerability(vuln, namespaceName))

			if vulnVersion != nil && vulnVersion.GreaterThan(fixedBy) {
				fixedBy = *vulnVersion
			}
		}
	}

	var fixedByStr string
	if fixedBy.GreaterThan(pkgVersion) {
		fixedByStr = fixedBy.String()
	}

	return fixedByStr, vulnerabilities
}

func rpmVersionPtr(ver rpmVersion.Version) *rpmVersion.Version {
	return &ver
}

// shareRepos takes repository definition and share it with other layers
// where repositories are missing.
// Returns a bool indicating if any CPEs exist.
func shareCPEs(layers []*database.RHELv2Layer) bool {
	var cpesExist bool

	// Users' layers built on top of Red Hat images don't have repository definitions.
	// We need to share CPE repo definitions to all layers where CPEs are missing.
	var previousCPEs []string
	for i := 0; i < len(layers); i++ {
		if len(layers[i].CPEs) != 0 {
			previousCPEs = layers[i].CPEs

			// Some layer has CPEs.
			cpesExist = true
		} else {
			layers[i].CPEs = append(layers[i].CPEs, previousCPEs...)
		}
	}

	// The same thing has to be done in reverse
	// example:
	//   Red Hat's base image doesn't have repository definition
	//   We need to get them from layer[i+1]
	for i := len(layers) - 1; i >= 0; i-- {
		if len(layers[i].CPEs) != 0 {
			previousCPEs = layers[i].CPEs
		} else {
			layers[i].CPEs = append(layers[i].CPEs, previousCPEs...)
		}
	}

	return cpesExist
}

// getRHELv2PkgEnvs returns a map from package ID to package environment and a bool to indicate CPEs exist in the image.
func getRHELv2PkgEnvs(db database.Datastore, layerName string) (map[int]*database.RHELv2PackageEnv, bool, error) {
	layers, err := db.GetRHELv2Layers(layerName)
	if err != nil {
		return nil, false, err
	}

	cpesExist := shareCPEs(layers)

	pkgEnvs := make(map[int]*database.RHELv2PackageEnv)

	// Find all packages which were ever added to the image,
	// labeled with the layer hash which introduced it.
	for _, layer := range layers {
		for _, pkg := range layer.Pkgs {
			if _, ok := pkgEnvs[pkg.ID]; !ok {
				pkgEnvs[pkg.ID] = &database.RHELv2PackageEnv{
					Pkg:       pkg,
					Namespace: layer.Dist,
					AddedBy:   layer.Hash,
					CPEs:      layer.CPEs,
				}
			}
		}
	}

	// Look for the packages which still remain in the final image.
	// Loop from the highest layer to base in search of the latest version of
	// the package database.
	for i := len(layers) - 1; i >= 0; i-- {
		if len(layers[i].Pkgs) == 0 {
			continue
		}

		// Found the latest version of `var/lib/rpm/Packages`
		// This has the final version of all the packages in this image.
		finalPkgs := set.NewIntSet()
		for _, pkg := range layers[i].Pkgs {
			finalPkgs.Add(pkg.ID)
		}

		for pkgID := range pkgEnvs {
			// Remove packages which were in lower layers, but not in the highest.
			if !finalPkgs.Contains(pkgID) {
				delete(pkgEnvs, pkgID)
			}
		}

		break
	}

	return pkgEnvs, cpesExist, nil
}

func getRHELv2Records(pkgEnvs map[int]*database.RHELv2PackageEnv) []*database.RHELv2Record {
	// Create a record for each pkgEnvironment for each CPE.
	var records []*database.RHELv2Record

	for _, pkgEnv := range pkgEnvs {
		if len(pkgEnv.CPEs) == 0 {
			records = append(records, &database.RHELv2Record{
				Pkg: pkgEnv.Pkg,
			})

			continue
		}

		for _, cpe := range pkgEnv.CPEs {
			records = append(records, &database.RHELv2Record{
				Pkg: pkgEnv.Pkg,
				CPE: cpe,
			})
		}
	}

	return records
}

// RHELv2ToVulnerability converts the given database.RHELv2Vulnerability into a Vulnerability.
func RHELv2ToVulnerability(vuln *database.RHELv2Vulnerability, namespace string) Vulnerability {
	var cvss2 types.MetadataCVSSv2
	if vuln.CVSSv2 != "" {
		scoreStr, vector := stringutils.Split2(vuln.CVSSv2, "/")
		score, err := strconv.ParseFloat(scoreStr, 64)
		if err != nil {
			log.Errorf("Unable to parse CVSSv2 score from RHEL vulnerability %s: %s", vuln.Name, vuln.CVSSv2)
		} else {
			cvss2Ptr, err := types.ConvertCVSSv2(vector)
			if err != nil {
				log.Errorf("Unable to parse CVSSv2 vector from RHEL vulnerability %s: %s", vuln.Name, vuln.CVSSv2)
			} else {
				if score != cvss2Ptr.Score {
					log.Warnf("Given CVSSv2 score and computed score differ for RHEL vulnerability %s: %f != %f. Using given score...", vuln.Name, score, cvss2Ptr.Score)
					cvss2Ptr.Score = score
				}

				cvss2 = *cvss2Ptr
			}
		}
	}

	var cvss3 types.MetadataCVSSv3
	if vuln.CVSSv3 != "" {
		scoreStr, vector := stringutils.Split2(vuln.CVSSv3, "/")
		score, err := strconv.ParseFloat(scoreStr, 64)
		if err != nil {
			log.Errorf("Unable to parse CVSSv3 score from RHEL vulnerability %s: %s", vuln.Name, vuln.CVSSv3)
		} else {
			cvss3Ptr, err := types.ConvertCVSSv3(vector)
			if err != nil {
				log.Errorf("Unable to parse CVSSv3 vector from RHEL vulnerability %s: %s", vuln.Name, vuln.CVSSv3)
			} else {
				if score != cvss3Ptr.Score {
					log.Warnf("Given CVSSv3 score and computed score differ for RHEL vulnerability %s: %f != %f. Using given score...", vuln.Name, score, cvss3Ptr.Score)
					cvss3Ptr.Score = score
				}

				cvss3 = *cvss3Ptr
			}
		}
	}

	var publishedTime, modifiedTime string
	if !vuln.Issued.IsZero() {
		publishedTime = vuln.Issued.Format(timeFormat)
	}
	if !vuln.Updated.IsZero() {
		modifiedTime = vuln.Updated.Format(timeFormat)
	}

	metadata := map[string]interface{}{
		"Red Hat": &types.Metadata{
			PublishedDateTime:    publishedTime,
			LastModifiedDateTime: modifiedTime,
			CVSSv2:               cvss2,
			CVSSv3:               cvss3,
		},
	}

	return Vulnerability{
		Name:          vuln.Name,
		NamespaceName: namespace,
		Description:   vuln.Description,
		Link:          vuln.Link,
		Severity:      vuln.Severity,
		Metadata:      metadata,
		// It is guaranteed there is 1 and only one element in `vuln.PackageInfos`.
		FixedBy: vuln.PackageInfos[0].FixedInVersion, // Empty string if not fixed.
	}
}
