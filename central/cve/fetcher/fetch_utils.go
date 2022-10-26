package fetcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/pkg/errors"
	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
)

const (
	fetchDelay            = 2 * time.Hour
	preloadedCVEsBasePath = "/stackrox/static-data"
	istioCVEsURL          = "https://definitions.stackrox.io/cve/istio/cve-list.json"
	istioCVEsChecksumURL  = "https://definitions.stackrox.io/cve/istio/checksum"
	commonCveDir          = "cve"
	istioCVEsDir          = "istio"
)

var (
	persistentCVEsPath                  = migrations.DBMountPath()
	persistentIstioCVEsFilePath         = filepath.Join(persistentCVEsPath, commonCveDir, istioCVEsDir, "cve-list.json")
	persistentIstioCVEsChecksumFilePath = filepath.Join(persistentCVEsPath, commonCveDir, istioCVEsDir, "checksum")
	preloadedIstioCVEsFilePath          = filepath.Join(preloadedCVEsBasePath, commonCveDir, nvd.Feeds[nvd.Istio].CVEFilename)
	preloadedIstioCVEsChecksumFilePath  = filepath.Join(preloadedCVEsBasePath, commonCveDir, nvd.Feeds[nvd.Istio].ChecksumFilename)

	log = logging.LoggerForModule()
)

func getLocalCVEChecksum(cveChecksumFile string) (string, error) {
	b, err := os.ReadFile(cveChecksumFile)
	if err != nil {
		return "", errors.Wrapf(err, "failed read k8s CVEs checksum file: %q", cveChecksumFile)
	}
	return string(b), nil
}

func getLocalCVEs(cveFile string) ([]*schema.NVDCVEFeedJSON10DefCVEItem, error) {
	b, err := os.ReadFile(cveFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed read CVEs file: %q", cveFile)
	}

	var cveEntries []*schema.NVDCVEFeedJSON10DefCVEItem
	if err = json.Unmarshal(b, &cveEntries); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal CVEs in file: %q", cveFile)
	}

	ret := make([]*schema.NVDCVEFeedJSON10DefCVEItem, 0, len(cveEntries))
	for i := 0; i < len(cveEntries); i++ {
		ret = append(ret, cveEntries[i])
	}

	return ret, nil
}

func overwriteCVEs(cveFile, cveChecksumFile, checksum, CVEs string) error {
	err := os.WriteFile(cveFile, []byte(CVEs), 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to overwrite CVEs file: %q", cveFile)
	}

	err = os.WriteFile(cveChecksumFile, []byte(checksum), 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to overwrite CVEs cveChecksumFile file: %q", cveChecksumFile)
	}

	return nil
}

func copyCVEsFromPreloadedToPersistentDirIfAbsent() error {
	paths := getIstioPaths()

	// Copying Istio CVE files
	if err := os.MkdirAll(paths.persistentCveDirPath, 0744); err != nil {
		log.Errorf("failed to create directory %q, err: %v", paths.persistentCveDirPath, err)
		return err
	}

	if err := fileutils.CopyNoOverwrite(paths.preloadedCveFile, paths.persistentCveFile); err != nil {
		return err
	}

	if err := fileutils.CopyNoOverwrite(paths.preloadedCveChecksumFile, paths.persistentCveChecksumFile); err != nil {
		return err
	}

	return nil
}

type cvePaths struct {
	preloadedCveDirPath       string
	persistentCveDirPath      string
	preloadedCveFile          string
	preloadedCveChecksumFile  string
	persistentCveFile         string
	persistentCveChecksumFile string
}

func getIstioPaths() *cvePaths {
	return &cvePaths{
		preloadedCveDirPath:       filepath.Join(preloadedCVEsBasePath, commonCveDir, istioCVEsDir),
		persistentCveDirPath:      filepath.Join(persistentCVEsPath, commonCveDir, istioCVEsDir),
		preloadedCveFile:          preloadedIstioCVEsFilePath,
		preloadedCveChecksumFile:  preloadedIstioCVEsChecksumFilePath,
		persistentCveFile:         persistentIstioCVEsFilePath,
		persistentCveChecksumFile: persistentIstioCVEsChecksumFilePath,
	}
}

type cveURLs struct {
	cveURL         string
	cveChecksumURL string
}

func getIstioUrls() *cveURLs {
	return &cveURLs{
		cveURL:         istioCVEsURL,
		cveChecksumURL: istioCVEsChecksumURL,
	}
}
