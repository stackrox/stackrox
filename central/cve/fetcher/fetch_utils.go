package fetcher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/pkg/errors"
	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	"github.com/stackrox/rox/central/cve/converter"
	licenseSingletons "github.com/stackrox/rox/central/license/singleton"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/license"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	fetchDelay            = 2 * time.Hour
	persistentCVEsPath    = migrations.DBMountPath
	preloadedCVEsBasePath = "/stackrox/data"
	k8sCVEsURL            = "https://definitions.stackrox.io/cve/k8s/cve-list.json"
	k8sCVEsChecksumURL    = "https://definitions.stackrox.io/cve/k8s/checksum"
	istioCVEsURL          = "https://definitions.stackrox.io/cve/istio/cve-list.json"
	istioCVEsChecksumURL  = "https://definitions.stackrox.io/cve/istio/checksum"
	commonCveDir          = "cve"
	k8sCVEsDir            = "k8s"
	istioCVEsDir          = "istio"
)

var (
	persistentK8sCVEsFilePath           = path.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir, "cve-list.json")
	persistentK8sCVEsChecksumFilePath   = path.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir, "checksum")
	preloadedK8sCVEsFilePath            = path.Join(preloadedCVEsBasePath, commonCveDir, nvd.Feeds[nvd.Kubernetes].CVEFilename)
	preloadedK8sCVEsChecksumFilePath    = path.Join(preloadedCVEsBasePath, commonCveDir, nvd.Feeds[nvd.Kubernetes].ChecksumFilename)
	persistentIstioCVEsFilePath         = path.Join(persistentCVEsPath, commonCveDir, istioCVEsDir, "cve-list.json")
	persistentIstioCVEsChecksumFilePath = path.Join(persistentCVEsPath, commonCveDir, istioCVEsDir, "checksum")
	preloadedIstioCVEsFilePath          = path.Join(preloadedCVEsBasePath, commonCveDir, nvd.Feeds[nvd.Istio].CVEFilename)
	preloadedIstioCVEsChecksumFilePath  = path.Join(preloadedCVEsBasePath, commonCveDir, nvd.Feeds[nvd.Istio].ChecksumFilename)
	log                                 = logging.LoggerForModule()

	cveTypeToString = map[converter.CVEType]string{
		converter.K8s:   "k8s",
		converter.Istio: "istio",
	}
)

func addLicenseIDAsQueryParam(baseURL string) (string, error) {
	licenseID, err := getCurrentLicenseID()
	if err != nil {
		return "", err
	}
	params := license.IDAsURLParam(licenseID)

	url, err := urlfmt.FullyQualifiedURL(baseURL, params)
	if err != nil {
		return "", err
	}
	return url, nil
}

func getCurrentLicenseID() (string, error) {
	license := licenseSingletons.ManagerSingleton().GetActiveLicense()

	if license == nil {
		return "", errors.New("active license not found")
	}
	return license.GetMetadata().GetId(), nil
}

func getLocalCVEChecksum(cveChecksumFile string) (string, error) {
	b, err := ioutil.ReadFile(cveChecksumFile)
	if err != nil {
		return "", errors.Wrapf(err, "failed read k8s CVEs checksum file: %q", cveChecksumFile)
	}
	return string(b), nil
}

func getLocalCVEs(cveFile string) ([]*schema.NVDCVEFeedJSON10DefCVEItem, error) {
	b, err := ioutil.ReadFile(cveFile)
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
	err := ioutil.WriteFile(cveFile, []byte(CVEs), 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to overwrite CVEs file: %q", cveFile)
	}

	err = ioutil.WriteFile(cveChecksumFile, []byte(checksum), 0644)
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
			preloadedCveDirPath:       path.Join(preloadedCVEsBasePath, commonCveDir, k8sCVEsDir),
			persistentCveDirPath:      path.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir),
			preloadedCveFile:          preloadedK8sCVEsFilePath,
			preloadedCveChecksumFile:  preloadedK8sCVEsChecksumFilePath,
			persistentCveFile:         persistentK8sCVEsFilePath,
			persistentCveChecksumFile: persistentK8sCVEsChecksumFilePath,
		}, nil
	} else if ct == converter.Istio {
		return &cvePaths{
			preloadedCveDirPath:       path.Join(preloadedCVEsBasePath, commonCveDir, istioCVEsDir),
			persistentCveDirPath:      path.Join(persistentCVEsPath, commonCveDir, istioCVEsDir),
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
