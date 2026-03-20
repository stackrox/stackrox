package index

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/quay/zlog"
	"github.com/rs/zerolog"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/internal/hostprobe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/utils"
)

func ConfigureClaircoreDebugLogging() {
	l := zerolog.New(os.Stderr).Level(zerolog.DebugLevel).With().Timestamp().Logger()
	zlog.Set(&l)
}

func logFilesystemDiagnostics(hostPath string) {
	switch hostprobe.DetectDNFVersion(hostPath) {
	case hostprobe.DNFVersion5:
		log.Info("DNF history DB (v5) found")
	case hostprobe.DNFVersion4:
		log.Info("DNF history DB (v4) found")
	default:
		log.Warn("DNF history DB not found!")
	}

	hasEntitlement, err := hostprobe.HasEntitlementCertKeyPair(hostPath)
	if err != nil {
		log.Warnf("Entitlement certificates not found: %v", err)
	} else if !hasEntitlement {
		log.Warn("Entitlement certificates not found")
	} else {
		log.Info("Entitlement certificates found")
	}

	allReposDirs := append(slices.Clone(hostprobe.DNF4ReposDirs), hostprobe.DNF5ReposDirPath)
	hasRepo, err := hostprobe.HasAnyRepoFile(os.DirFS(hostPath), allReposDirs)
	if err != nil {
		logRepoError(err)
		return
	}
	if !hasRepo {
		log.Info("Repo dirs are present but contain 0 .repo files")
		return
	}
	log.Info("Repo dirs contain .repo files")
}

func logRepoError(err error) {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		log.Info("No repo directories found")
	case errors.Is(err, fs.ErrPermission):
		log.Warnf("Repo directories are not readable: %v", err)
	default:
		log.Infof("Repo directories are unavailable: %v", err)
	}
}

func logIndexReportDiagnostics(report *v4.IndexReport, debug bool) {
	contents := report.GetContents()

	numPkgs := len(contents.GetPackages())
	numRepos := len(contents.GetRepositories())
	numDists := len(contents.GetDistributions())
	numEnvs := len(contents.GetEnvironments())

	log.Infof("Index report summary: packages=%d, repositories=%d, distributions=%d, environments=%d",
		numPkgs, numRepos, numDists, numEnvs)

	repos := contents.GetRepositories()
	for repoIdx, id := range slices.Sorted(maps.Keys(repos)) {
		repo := repos[id]
		log.Infof("Repository (%d of %d) id=%q name=%q key=%q cpe=%q",
			repoIdx+1, numRepos, id, repo.GetName(), repo.GetKey(), repo.GetCpe())
		if !debug && repoIdx+1 >= 10 && numRepos > 10 {
			log.Infof("  (%d more truncated for brevity)", numRepos-10)
			break
		}
	}
	dists := contents.GetDistributions()
	for distIdx, id := range slices.Sorted(maps.Keys(dists)) {
		dist := dists[id]
		log.Infof("Distribution (%d of %d) id=%s name=%q version=%q cpe=%q did=%q",
			distIdx+1, numDists, id, dist.GetName(), dist.GetVersion(), dist.GetCpe(), dist.GetDid())
		if !debug && distIdx+1 >= 10 && numDists > 10 {
			log.Infof("  (%d more truncated for brevity)", numDists-10)
			break
		}
	}

	if numRepos == 0 {
		log.Warn("Index report contains 0 repositories. Packages will be marked as UNSCANNED.")
	}
}

// fetchRepo2CPEMappingForDiagnostics fetches the repo2cpe mapping to check if it is available and not empty.
// The downloaded result is discarded as with current API, it cannot be provided to the matcher as an input.
func fetchRepo2CPEMappingForDiagnostics(ctx context.Context, mappingURL string, timeout time.Duration, client *http.Client) {
	if mappingURL == "" {
		log.Warn("Repo2CPE mapping URL is empty")
		return
	}
	log.Info("Attempting to fetch repo2cpe mapping for diagnostics...")

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, mappingURL, nil)
	if err != nil {
		log.Warnf("Could not create repo2cpe mapping request for %q: %v", mappingURL, err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Warnf("Could not download repo2cpe mapping from %q: %v", mappingURL, err)
		return
	}
	defer utils.IgnoreError(resp.Body.Close)
	size, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		log.Warnf("Could not read repo2cpe mapping response body from %q: %v", mappingURL, err)
		return
	}

	log.Debugf("Repo2CPE mapping fetch: status=%d, size=%d bytes", resp.StatusCode, size)
	switch {
	case resp.StatusCode >= http.StatusBadRequest || size == 0:
		log.Warnf("Repo2CPE mapping is unavailable or empty (status=%d, size=%d)", resp.StatusCode, size)
	case resp.StatusCode == http.StatusOK:
		log.Info("Repo2CPE mapping downloaded successfully")
	default:
		log.Infof("Repo2CPE mapping response: status=%d, size=%d bytes", resp.StatusCode, size)
	}
}
