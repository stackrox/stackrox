package fetcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/pkg/errors"
	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	"github.com/stackrox/stackrox/central/cve/converter"
	"github.com/stackrox/stackrox/pkg/fileutils"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/migrations"
)

const (
	fetchDelay            = 2 * time.Hour
	preloadedCVEsBasePath = "/stackrox/static-data"
	k8sCVEsURL            = "https://definitions.stackrox.io/cve/k8s/cve-list.json"
	k8sCVEsChecksumURL    = "https://definitions.stackrox.io/cve/k8s/checksum"
	istioCVEsURL          = "https://definitions.stackrox.io/cve/istio/cve-list.json"
	istioCVEsChecksumURL  = "https://definitions.stackrox.io/cve/istio/checksum"
	commonCveDir          = "cve"
	k8sCVEsDir            = "k8s"
	istioCVEsDir          = "istio"
)

var (
	persistentCVEsPath                  = migrations.DBMountPath()
	persistentK8sCVEsFilePath           = filepath.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir, "cve-list.json")
	persistentK8sCVEsChecksumFilePath   = filepath.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir, "checksum")
	preloadedK8sCVEsFilePath            = filepath.Join(preloadedCVEsBasePath, commonCveDir, nvd.Feeds[nvd.Kubernetes].CVEFilename)
	preloadedK8sCVEsChecksumFilePath    = filepath.Join(preloadedCVEsBasePath, commonCveDir, nvd.Feeds[nvd.Kubernetes].ChecksumFilename)
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

func copyCVEsFromPreloadedToPersistentDirIfAbsent(ct converter.CVEType) error {
	paths, err := getPaths(ct)
	if err != nil {
		return err
	}

	// Copying k8s CVE files
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

func getPaths(ct converter.CVEType) (*cvePaths, error) {
	if ct == converter.K8s {
		return &cvePaths{
			preloadedCveDirPath:       filepath.Join(preloadedCVEsBasePath, commonCveDir, k8sCVEsDir),
			persistentCveDirPath:      filepath.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir),
			preloadedCveFile:          preloadedK8sCVEsFilePath,
			preloadedCveChecksumFile:  preloadedK8sCVEsChecksumFilePath,
			persistentCveFile:         persistentK8sCVEsFilePath,
			persistentCveChecksumFile: persistentK8sCVEsChecksumFilePath,
		}, nil
	} else if ct == converter.Istio {
		return &cvePaths{
			preloadedCveDirPath:       filepath.Join(preloadedCVEsBasePath, commonCveDir, istioCVEsDir),
			persistentCveDirPath:      filepath.Join(persistentCVEsPath, commonCveDir, istioCVEsDir),
			preloadedCveFile:          preloadedIstioCVEsFilePath,
			preloadedCveChecksumFile:  preloadedIstioCVEsChecksumFilePath,
			persistentCveFile:         persistentIstioCVEsFilePath,
			persistentCveChecksumFile: persistentIstioCVEsChecksumFilePath,
		}, nil
	} else {
		return &cvePaths{}, fmt.Errorf("unknown cve type: %d", ct)
	}
}

type cveURLs struct {
	cveURL         string
	cveChecksumURL string
}

func getUrls(ct converter.CVEType) (*cveURLs, error) {
	if ct == converter.K8s {
		return &cveURLs{
			cveURL:         k8sCVEsURL,
			cveChecksumURL: k8sCVEsChecksumURL,
		}, nil
	} else if ct == converter.Istio {
		return &cveURLs{
			cveURL:         istioCVEsURL,
			cveChecksumURL: istioCVEsChecksumURL,
		}, nil
	} else {
		return &cveURLs{}, fmt.Errorf("unknown cve type: %d", ct)
	}
}
