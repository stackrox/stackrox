package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/cve/utils"
	licenseSingletons "github.com/stackrox/rox/central/license/singleton"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
)

const (
	fetchDelay               = 2 * time.Hour
	persistentCVEsPath       = migrations.DBMountPath
	preloadedCVEsBasePath    = "/stackrox/data"
	k8sCVEsURL               = "https://definitions.stackrox.io/cve/k8s/cve-list.json"
	k8sCVEsChecksumURL       = "https://definitions.stackrox.io/cve/k8s/checksum"
	istioCVEsURL             = "https://definitions.stackrox.io/cve/istio/cve-list.json"
	istioCVEsChecksumURL     = "https://definitions.stackrox.io/cve/istio/checksum"
	commonCveDir             = "cve"
	k8sCVEsDir               = "k8s"
	istioCVEsDir             = "istio"
	scannerDefinitionsSubdir = `scannerdefinitions`
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

	cveTypeToString = map[converter.CveType]string{
		converter.K8s:   "k8s",
		converter.Istio: "istio",
	}
)

func fetchRemote(baseURL string) (string, error) {
	url, err := addLicenseIDAsQueryParam(baseURL)
	if err != nil {
		return "", err
	}

	resp, err := utils.RunHTTPGet(url)
	if err != nil {
		return "", err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Error(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		buf, err := utils.ReadNBytesFromResponse(resp, 1024)
		if err != nil {
			return "", errors.Wrapf(err, "failed to download from %q, additionally, there was an error reading the response body. status code: %d, status: %s", url, resp.StatusCode, resp.Status)
		}
		return "", fmt.Errorf("failed to download from %q. status code: %d, status: %s, response body: %s", url, resp.StatusCode, resp.Status, string(buf))
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "got HTTP response %d, but failed to read from response body", resp.StatusCode)
	}
	log.Infof("successful fetch from %s, bytes read : %d", url, len(b))

	return string(b), nil
}

func addLicenseIDAsQueryParam(baseURL string) (string, error) {
	licenseID, err := getCurrentLicenseID()
	if err != nil {
		return "", err
	}

	queryParams := []utils.QueryParam{
		{
			Key:   "license_id",
			Value: licenseID,
		},
	}

	url, err := utils.GetURLWithQueryParams(baseURL, queryParams)
	if err != nil {
		return "", err
	}
	return url, nil
}

func getCurrentLicenseID() (string, error) {
	licenseMgr := licenseSingletons.ManagerSingleton()
	license := licenseMgr.GetActiveLicense()

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

func getLocalCVEs(cveFile string) ([]*nvd.CVEEntry, error) {
	b, err := ioutil.ReadFile(cveFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed read CVEs file: %q", cveFile)
	}

	var cveEntries []nvd.CVEEntry
	if err = json.Unmarshal(b, &cveEntries); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal CVEs in file: %q", cveFile)
	}

	ret := make([]*nvd.CVEEntry, 0, len(cveEntries))
	for i := 0; i < len(cveEntries); i++ {
		ret = append(ret, &cveEntries[i])
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

func copyCVEsFromPreloadedToPersistentDirIfAbsent(ct converter.CveType) error {
	paths, err := getPaths(ct)
	if err != nil {
		return err
	}

	// Copying k8s CVE files
	if err := os.MkdirAll(paths.persistentCveDirPath, 0744); err != nil {
		log.Errorf("failed to create directory %q, err: %v", paths.persistentCveDirPath, err)
		return err
	}

	if err := copyFileIfAbsent(paths.preloadedCveFile, paths.persistentCveFile); err != nil {
		return err
	}

	if err := copyFileIfAbsent(paths.preloadedCveChecksumFile, paths.persistentCveChecksumFile); err != nil {
		return err
	}

	return nil
}

func copyFileIfAbsent(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return errors.Wrapf(err, "failed to open src file: %q", src)
	}
	defer func() {
		err := in.Close()
		if err != nil {
			log.Error(err)
		}
	}()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to open dst file: %q", dst)
	}
	defer func() {
		err := out.Close()
		if err != nil {
			log.Error(err)
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return errors.Wrapf(err, "failed to copy src: %q to dst: %q", src, dst)
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

func getPaths(ct converter.CveType) (*cvePaths, error) {
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

func getUrls(ct converter.CveType) (*cveURLs, error) {
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
