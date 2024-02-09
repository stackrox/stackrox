package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/quay/clair-action/datastore"
	"github.com/quay/claircore"
	"github.com/quay/claircore/enricher/cvss"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/zlog"
	"github.com/rs/zerolog"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/utils"
)

func (i *imageScanCommand) localScan() error {
	imageResult, err := i.scanLocal(context.TODO())
	if err != nil {
		return err
	}
	return i.printImageResult(imageResult)
}

func (i *imageScanCommand) scanLocal(ctx context.Context) (*storage.Image, error) {
	nop := zerolog.Nop()
	zlog.Set(&nop)

	dbPath = "/home/janisz/go/src/github.com/stackrox/clair-action/vulndb"
	dbURL = "https://clair-sqlite-db.s3.amazonaws.com/matcher.zst"

	fa := &LocalFetchArena{}

	img, err := newImage(i.image)
	if err != nil {
		return nil, fmt.Errorf("could not get image to scan: %w", err)
	}
	defer os.Remove(img.path)

	//err := datastore.DownloadDB(ctx, dbURL, dbPath)
	//if err != nil {
	//	return nil, fmt.Errorf("could not download database: %w", err)
	//}

	println("Creating sqlite matcher store")

	matcherStore, err := datastore.NewSQLiteMatcherStore(dbPath, true)
	if err != nil {
		return nil, fmt.Errorf("error creating sqlite backend: %w", err)
	}

	matcherOpts := &libvuln.Options{
		Store:                    matcherStore,
		DisableBackgroundUpdates: true,
		UpdateWorkers:            1,
		Enrichers: []driver.Enricher{
			&cvss.Enricher{},
		},
		Client: http.DefaultClient,
	}
	println("Creating libvuln")
	lv, err := libvuln.New(ctx, matcherOpts)
	if err != nil {
		return nil, fmt.Errorf("error creating Libvuln: %w", err)
	}

	println("Get manifest")
	mf, err := img.GetManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating manifest: %w", err)
	}

	indexerOpts := &libindex.Options{
		Store:      datastore.NewLocalIndexerStore(),
		Locker:     updates.NewLocalLockSource(),
		FetchArena: fa,
	}
	println("Creating new index")
	li, err := libindex.New(ctx, indexerOpts, http.DefaultClient)
	if err != nil {
		return nil, fmt.Errorf("error creating Libindex: %w", err)
	}
	println("Indexing...")
	ir, err := li.Index(ctx, mf)
	utils.Must(err)

	println("Scanning...")
	vr, err := lv.Scan(ctx, ir)
	if err != nil {
		return nil, fmt.Errorf("error creating vulnerability report: %w", err)
	}

	return i.convertToImage(vr, img.image), nil
}

func (i *imageScanCommand) convertToImage(vr *claircore.VulnerabilityReport, image v1.Image) *storage.Image {
	components := []*storage.EmbeddedImageScanComponent{}
	fixable := 0
	for _, v := range vr.Vulnerabilities {
		if v.FixedInVersion != "" {
			fixable += 1
		}
	}
	totalTop := 0.0
	for id, pkg := range vr.Packages {
		top := 0.0
		vulns := []*storage.EmbeddedVulnerability{}
		for _, vuln := range vr.PackageVulnerabilities[id] {
			v := vr.Vulnerabilities[vuln]

			const enrichmentMap = "message/vnd.clair.map.vulnerability; enricher=clair.cvss schema=https://csrc.nist.gov/schema/nvd/feed/1.1/cvss-v3.x.json"

			enrichment := &Enrichment{}
			enrichmentsObj, ok := vr.Enrichments[enrichmentMap]
			if ok {
				ens := map[string][]*Enrichment{}
				err := json.Unmarshal(enrichmentsObj[0], &ens)
				utils.Must(err)
				enrichments, ok := ens[vuln]
				if ok {
					enrichment = getMostSevereEnrichment(enrichments)
					// Reclassify severity using the CVSS severity if Unknown
					if v.Severity == "Unknown" {
						v.Severity = strings.Title(strings.ToLower(enrichment.BaseSeverity))
					}
				}
			}

			top = math.Max(top, float64(enrichment.BaseScore))

			vulns = append(vulns, &storage.EmbeddedVulnerability{
				Cve:          v.Name,
				Cvss:         enrichment.BaseScore,
				Summary:      v.Description,
				Link:         v.Links,
				SetFixedBy:   &storage.EmbeddedVulnerability_FixedBy{FixedBy: v.FixedInVersion},
				ScoreVersion: storage.EmbeddedVulnerability_V3,
				CvssV3: &storage.CVSSV3{
					Vector:              enrichment.VectorString,
					ExploitabilityScore: enrichment.BaseScore,
					ImpactScore:         enrichment.BaseScore,
					AttackVector:        storage.CVSSV3_AttackVector(storage.CVSSV3_AttackVector_value["ATTACK_"+enrichment.AttackVector]),
					AttackComplexity:    storage.CVSSV3_Complexity(storage.CVSSV3_Complexity_value["COMPLEXITY_"+enrichment.AttackComplexity]),
					PrivilegesRequired:  storage.CVSSV3_Privileges(storage.CVSSV3_Privileges_value["PRIVILEGE_"+enrichment.PrivilegesRequired]),
					UserInteraction:     storage.CVSSV3_UserInteraction(storage.CVSSV3_UserInteraction_value["UI_"+enrichment.UserInteraction]),
					Scope:               storage.CVSSV3_Scope(storage.CVSSV3_Scope_value[enrichment.Scope]),
					Confidentiality:     storage.CVSSV3_Impact(storage.CVSSV3_Impact_value["IMPACT_"+enrichment.ConfidentialityImpact]),
					Integrity:           storage.CVSSV3_Impact(storage.CVSSV3_Impact_value["IMPACT_"+enrichment.IntegrityImpact]),
					Availability:        storage.CVSSV3_Impact(storage.CVSSV3_Impact_value["IMPACT_"+enrichment.AvailabilityImpact]),
					Score:               enrichment.BaseScore,
					Severity:            storage.CVSSV3_Severity(storage.CVSSV3_Severity_value[enrichment.BaseSeverity]),
				},
				PublishedOn:           protoconv.ConvertTimeToTimestamp(v.Issued),
				LastModified:          protoconv.ConvertTimeToTimestamp(v.Issued),
				VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
				VulnerabilityTypes:    nil,
				Suppressed:            false,
				SuppressActivation:    nil,
				SuppressExpiry:        nil,
				FirstSystemOccurrence: nil,
				FirstImageOccurrence:  nil,
				Severity:              storage.VulnerabilitySeverity(storage.VulnerabilitySeverity_value[enrichment.BaseSeverity+"_VULNERABILITY_SEVERITY"]),
				State:                 storage.VulnerabilityState_OBSERVED,
			})

		}

		components = append(components, &storage.EmbeddedImageScanComponent{
			Name:     pkg.Name,
			Version:  pkg.Version,
			Vulns:    vulns,
			Location: pkg.Filepath,
			SetTopCvss: &storage.EmbeddedImageScanComponent_TopCvss{
				TopCvss: float32(top),
			},
			FixedBy: "",
		})
		totalTop = math.Max(top, totalTop)
	}

	digest, err := image.Digest()
	utils.Must(err)

	result := &storage.Image{
		Id: digest.String(),
		Name: &storage.ImageName{
			FullName: i.image,
		},
		Names:    nil,
		Metadata: nil,
		Scan: &storage.ImageScan{
			ScannerVersion:  "local",
			ScanTime:        protoconv.ConvertTimeToTimestamp(time.Now()),
			Components:      components,
			OperatingSystem: runtime.GOOS,
			DataSource:      nil,
			Notes:           nil,
			Hashoneof:       nil,
		},
		SignatureVerificationData: nil,
		Signature:                 nil,
		SetComponents:             &storage.Image_Components{Components: int32(len(components))},
		SetCves:                   &storage.Image_Cves{Cves: int32(len(vr.Vulnerabilities))},
		SetFixable:                &storage.Image_FixableCves{FixableCves: int32(fixable)},
		LastUpdated:               protoconv.ConvertTimeToTimestamp(time.Now()),
		NotPullable:               false,
		IsClusterLocal:            false,
		Priority:                  0,
		RiskScore:                 0,
		SetTopCvss:                &storage.Image_TopCvss{TopCvss: float32(totalTop)},
		Notes:                     nil,
	}
	//bar.Finish()
	return result
}

type Enrichment struct {
	Version               string  `json:"version"`
	VectorString          string  `json:"vectorString"`
	AttackVector          string  `json:"attackVector"`
	AttackComplexity      string  `json:"attackComplexity"`
	PrivilegesRequired    string  `json:"privilegesRequired"`
	UserInteraction       string  `json:"userInteraction"`
	Scope                 string  `json:"scope"`
	ConfidentialityImpact string  `json:"confidentialityImpact"`
	IntegrityImpact       string  `json:"integrityImpact"`
	AvailabilityImpact    string  `json:"availabilityImpact"`
	BaseSeverity          string  `json:"baseSeverity"`
	BaseScore             float32 `json:"baseScore"`
}

// GetMostSevereEnrichment will take a slice of Enrichments and return
// the one with the highest baseScore.
//
// This is done to mimic how quay displays this data, future enhancements
// should present all enrichment data keyed by CVE.
func getMostSevereEnrichment(enrichments []*Enrichment) *Enrichment {
	if len(enrichments) == 1 {
		return enrichments[0]
	}
	severeEnrichment := enrichments[0]
	for i := 1; i < len(enrichments)-1; i++ {
		if enrichments[i].BaseScore > severeEnrichment.BaseScore {
			severeEnrichment = enrichments[i]
		}
	}
	return severeEnrichment
}
