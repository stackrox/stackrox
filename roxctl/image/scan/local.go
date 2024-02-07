package scan

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/clair-action/datastore"
	"github.com/quay/claircore/enricher/cvss"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/pkg/tarfs"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
)

func (i *imageScanCommand) localScan() error {
	imageResult, err := i.scanLocal()
	if err != nil {
		return err
	}
	return i.printImageResult(imageResult)
}

func (i *imageScanCommand) scanLocal() (*storage.Image, error) {
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), i.timeout)
	defer cancel()

	var (
		// All arguments
		imgRef = i.image
		//imgPath         = c.String("image-path")
		dbPath = "/home/janisz/go/src/github.com/stackrox/clair-action/vulndb"
		//dbURL  = "https://clair-sqlite-db.s3.amazonaws.com/matcher.zst"
		//format          = c.String("format")
		//returnCode      = c.Int("return-code")
		//dockerConfigDir = c.String("docker-config-dir")
	)

	fa := &LocalFetchArena{}
	img, err := NewImage(imgRef)
	if err != nil {
		return nil, fmt.Errorf("could not get image to scan: %w", err)
	}

	//err := datastore.DownloadDB(ctx, dbURL, dbPath)
	//if err != nil {
	//	return nil, fmt.Errorf("could not download database: %w", err)
	//}

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
	lv, err := libvuln.New(ctx, matcherOpts)
	if err != nil {
		return nil, fmt.Errorf("error creating Libvuln: %w", err)
	}

	mf, err := img.GetManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating manifest: %w", err)
	}

	indexerOpts := &libindex.Options{
		Store:      datastore.NewLocalIndexerStore(),
		Locker:     updates.NewLocalLockSource(),
		FetchArena: fa,
	}
	li, err := libindex.New(ctx, indexerOpts, http.DefaultClient)
	if err != nil {
		return nil, fmt.Errorf("error creating Libindex: %w", err)
	}
	ir, err := li.Index(ctx, mf)
	// TODO (crozzy) Better error handling once claircore
	// error overhaul is merged.
	switch {
	case errors.Is(err, nil):
	case errors.Is(err, tarfs.ErrFormat):
		return nil, fmt.Errorf("error creating index report due to invalid layer: %w", err)
	default:
		return nil, fmt.Errorf("error creating index report: %w", err)
	}

	vr, err := lv.Scan(ctx, ir)
	if err != nil {
		return nil, fmt.Errorf("error creating vulnerability report: %w", err)
	}

	components := []*storage.EmbeddedImageScanComponent{}
	for id, pkg := range vr.Packages {
		vulns := []*storage.EmbeddedVulnerability{}
		for _, vuln := range vr.PackageVulnerabilities[id] {
			v := vr.Vulnerabilities[vuln]
			vulns = append(vulns, &storage.EmbeddedVulnerability{
				Cve:         v.Name,
				Summary:     v.Description,
				Link:        v.Links,
				SetFixedBy:  &storage.EmbeddedVulnerability_FixedBy{FixedBy: v.FixedInVersion},
				PublishedOn: protoconv.ConvertTimeToTimestamp(v.Issued),
			})
		}

		components = append(components, &storage.EmbeddedImageScanComponent{
			Name:       pkg.Name,
			Version:    pkg.Version,
			Vulns:      vulns,
			Location:   pkg.Filepath,
			SetTopCvss: nil,
			FixedBy:    "",
		})
	}

	result := &storage.Image{
		Id: vr.Hash.String(),
		Name: &storage.ImageName{
			Registry: "",
			Remote:   "",
			Tag:      "",
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
		SetComponents:             nil,
		SetCves:                   nil,
		SetFixable:                nil,
		LastUpdated:               nil,
		NotPullable:               false,
		IsClusterLocal:            false,
		Priority:                  0,
		RiskScore:                 0,
		SetTopCvss:                nil,
		Notes:                     nil,
	}

	return result, nil
}
