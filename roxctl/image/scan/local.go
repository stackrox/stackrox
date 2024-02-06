package scan

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/clair-action/datastore"
	"github.com/quay/clair-action/image"
	"github.com/quay/claircore/enricher/cvss"
	"github.com/quay/claircore/indexer"
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

	var (
		img image.Image
		fa  indexer.FetchArena
	)

	if _, err := os.Stat(i.image); errors.Is(err, os.ErrNotExist) {
		img = image.NewDockerRemoteImage(ctx, imgRef)
	} else {
		var err error
		img, err = image.NewDockerLocalImage(ctx, i.image, os.TempDir())
		if err != nil {
			return fmt.Errorf("error getting image information: %v", err)
		}
	}

	//err := datastore.DownloadDB(ctx, dbURL, dbPath)
	//if err != nil {
	//	return fmt.Errorf("could not download database: %v", err)
	//}

	matcherStore, err := datastore.NewSQLiteMatcherStore(dbPath, true)
	if err != nil {
		return fmt.Errorf("error creating sqlite backend: %v", err)
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
		return fmt.Errorf("error creating Libvuln: %v", err)
	}

	mf, err := img.GetManifest(ctx)
	if err != nil {
		return fmt.Errorf("error creating manifest: %v", err)
	}

	fa = libindex.NewRemoteFetchArena(http.DefaultClient, os.TempDir())
	indexerOpts := &libindex.Options{
		Store:      datastore.NewLocalIndexerStore(),
		Locker:     updates.NewLocalLockSource(),
		FetchArena: fa,
	}
	li, err := libindex.New(ctx, indexerOpts, http.DefaultClient)
	if err != nil {
		return fmt.Errorf("error creating Libindex: %v", err)
	}
	ir, err := li.Index(ctx, mf)
	// TODO (crozzy) Better error handling once claircore
	// error overhaul is merged.
	switch {
	case errors.Is(err, nil):
	case errors.Is(err, tarfs.ErrFormat):
		return fmt.Errorf("error creating index report due to invalid layer: %v", err)
	default:
		return fmt.Errorf("error creating index report: %v", err)
	}

	vr, err := lv.Scan(ctx, ir)
	if err != nil {
		return fmt.Errorf("error creating vulnerability report: %v", err)
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

	return i.printImageResult(result)
}
