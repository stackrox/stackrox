package nvdtoolscache

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd"
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/scanner/pkg/vulnloader/nvdloader"
	"github.com/stackrox/scanner/pkg/ziputil"
)

func (c *cacheImpl) LoadFromDirectory(definitionsDir string) error {
	log.WithField("dir", definitionsDir).Info("Loading definitions directory")

	files, err := os.ReadDir(definitionsDir)
	if err != nil {
		return err
	}

	var totalVulns int
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		numVulns, err := c.handleJSONFile(filepath.Join(definitionsDir, f.Name()))
		if err != nil {
			return errors.Wrapf(err, "handling file %s", f.Name())
		}
		totalVulns += numVulns
	}
	log.Infof("Total vulns in %q: %d", definitionsDir, totalVulns)

	utils.Must(c.sync())
	return nil
}

func (c *cacheImpl) LoadFromZip(zipR *zip.ReadCloser, definitionsDir string) error {
	log.WithField("dir", definitionsDir).Info("Loading definitions directory")

	readers, err := ziputil.OpenFilesInDir(zipR, definitionsDir, ".json")
	if err != nil {
		return err
	}

	var totalVulns int
	for _, r := range readers {
		numVulns, err := c.handleReader(r)
		if err != nil {
			return errors.Wrapf(err, "handling file %s", r.Name)
		}
		totalVulns += numVulns
	}
	log.Infof("Total vulns in %s: %d", definitionsDir, totalVulns)

	utils.Must(c.sync())
	return nil
}

func cpeIsApplicationOrLinuxKernel(cpe string) bool {
	spl := strings.SplitN(cpe, ":", 6)
	if len(spl) < 6 {
		return false
	}
	// Check if the application is valid.
	// Empty or ANY product is not valid.
	if spl[2] == "a" && spl[4] != "" && spl[4] != "*" {
		return true
	}

	// Return true if this is a linux kernel CPE.
	return spl[2] == "o" && spl[3] == "linux" && spl[4] == "linux_kernel"
}

func isNodeValid(node *schema.NVDCVEFeedJSON10DefNode) bool {
	if len(node.CPEMatch) != 0 {
		filteredCPEs := node.CPEMatch[:0]
		for _, cpe := range node.CPEMatch {
			if cpeIsApplicationOrLinuxKernel(cpe.Cpe23Uri) {
				filteredCPEs = append(filteredCPEs, cpe)
			}
		}
		node.CPEMatch = filteredCPEs
		return len(filteredCPEs) != 0
	}
	// Otherwise look at the children and make sure if the Operator is an AND they are all valid
	if strings.EqualFold(node.Operator, "and") {
		for _, c := range node.Children {
			if !isNodeValid(c) {
				return false
			}
		}
		return true
	}
	// Operator is an OR so ensure at least one is valid
	filteredNodes := node.Children[:0]
	for _, c := range node.Children {
		if isNodeValid(c) {
			filteredNodes = append(filteredNodes, c)
		}
	}
	node.Children = filteredNodes
	return len(filteredNodes) != 0
}

func isValidCVE(cve *schema.NVDCVEFeedJSON10DefCVEItem) bool {
	if cve.Configurations == nil {
		return false
	}
	filteredNodes := cve.Configurations.Nodes[:0]
	for _, n := range cve.Configurations.Nodes {
		if isNodeValid(n) {
			filteredNodes = append(filteredNodes, n)
		}
	}
	cve.Configurations.Nodes = filteredNodes
	return len(filteredNodes) != 0
}

func trimCVE(cve *schema.NVDCVEFeedJSON10DefCVEItem) {
	cve.CVE.References = nil
	cve.CVE.Affects = nil
	cve.CVE.DataType = ""
	cve.CVE.Problemtype = nil
	cve.CVE.DataVersion = ""
	cve.CVE.DataFormat = ""
	cve.Configurations.CVEDataVersion = ""
}

func (c *cacheImpl) handleJSONFile(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, errors.Wrapf(err, "opening file at %q", path)
	}

	return c.handleReader(&ziputil.ReadCloser{
		ReadCloser: f,
		Name:       path,
	})
}

// handleReader loads the given reader and closes it when finished.
func (c *cacheImpl) handleReader(r *ziputil.ReadCloser) (int, error) {
	defer utils.IgnoreError(r.Close)

	feed, err := nvdloader.LoadJSONFileFromReader(r)
	if err != nil {
		return 0, errors.Wrapf(err, "loading JSON file at path %q", r.Name)
	}

	var numVulns int
	for _, cve := range feed.CVEItems {
		if cve == nil || cve.Configurations == nil {
			continue
		}
		if !isValidCVE(cve) {
			continue
		}

		vuln := nvd.ToVuln(cve)
		trimCVE(cve)

		err := c.addProductToCVE(vuln, cve)
		if err != nil {
			return 0, errors.Wrapf(err, "adding vuln %q from %q", vuln.ID(), r.Name)
		}

		numVulns++
	}
	return numVulns, nil
}
