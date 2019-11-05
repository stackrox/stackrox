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
	"github.com/stackrox/rox/central/cve/utils"
	licenseSingletons "github.com/stackrox/rox/central/license/singleton"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
)

const (
	fetchDelay            = 2 * time.Hour
	persistentCVEsPath    = migrations.DBMountPath
	preloadedCVEsBasePath = "/stackrox/data"
	k8sCVEsURL            = "https://definitions.stackrox.io/cve/k8s/cve-list.json"
	k8sCVEsHashURL        = "https://definitions.stackrox.io/cve/k8s/checksum"
	commonCveDir          = "cve"
	k8sCVEsDir            = "k8s"
)

var (
	k8sCVEsPersistentFilePath     = path.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir, "cve-list.json")
	k8sCVEsHashPersistentFilePath = path.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir, "cve-cve_checksum")
	k8sCVEsEphemeralFilePath      = path.Join(preloadedCVEsBasePath, commonCveDir, nvd.Feeds[nvd.Kubernetes].CVEFilename)
	k8sCVEsHashEphemeralFilePath  = path.Join(preloadedCVEsBasePath, commonCveDir, nvd.Feeds[nvd.Kubernetes].ChecksumFilename)

	log = logging.LoggerForModule()
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

func getLocalCVEsHash(cveHashFile string) (string, error) {
	b, err := ioutil.ReadFile(k8sCVEsHashPersistentFilePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed read k8s CVEs hash file: %q", cveHashFile)
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

func overwriteCVEs(cveFile, cveHashFile, hash, CVEs string) error {
	err := ioutil.WriteFile(cveFile, []byte(CVEs), 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to overwrite CVEs file: %q", cveFile)
	}

	err = ioutil.WriteFile(cveHashFile, []byte(hash), 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to overwrite k8s CVEs hash file: %q", cveHashFile)
	}

	return nil
}

func copyK8sCVEsFromPreloadedToPersistentDirIfAbsent() error {
	persistentDirPath := path.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir)
	if err := os.MkdirAll(persistentDirPath, 0744); err != nil {
		log.Errorf("failed to create directory %q, err: %v", persistentDirPath, err)
		return err
	}

	if err := copyFileIfAbsent(k8sCVEsEphemeralFilePath, k8sCVEsPersistentFilePath); err != nil {
		return err
	}

	if err := copyFileIfAbsent(k8sCVEsHashEphemeralFilePath, k8sCVEsHashPersistentFilePath); err != nil {
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
