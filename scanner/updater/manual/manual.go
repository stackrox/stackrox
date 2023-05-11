package manual

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/url"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
)

// Factory is the UpdaterSetFactory exposed by this package.
//
// All configuration is done on the returned updaters. See the [Config] type.
var Factory driver.UpdaterSetFactory = &factory{}

type factory struct{}

func (factory) UpdaterSet(context.Context) (s driver.UpdaterSet, err error) {
	s = driver.NewUpdaterSet()
	s.Add(&updater{})
	return s, nil
}

type updater struct {
	c    *http.Client
	root *url.URL
	// Allow is a bool-and-map-of-bool.
	//
	// If populated, only extant entries are allowed. If not populated,
	// everything is allowed. It uses a bool to make a conditional simpler later.
	allow map[string]bool
}

// Config is the configuration that this updater accepts.
//
// By convention, it's at a key called "osv".
type Config struct {
	// The URL serving data dumps behind an S3 API.
	//
	// Authentication is unconfigurable, the ListObjectsV2 API must be publicly
	// accessible.
	URL string `json:"url" yaml:"url"`
	// Allowlist is a list of ecosystems to allow. When this is unset, all are
	// allowed.
	//
	// Extant ecosystems are discovered at runtime, see the OSV Schema
	// (https://ossf.github.io/osv-schema/) or the "ecosystems.txt" file in the
	// OSV data for the current list.
	Allowlist []string `json:"allowlist" yaml:"allowlist"`
}

var _ driver.Updater = (*updater)(nil)

func (u *updater) Name() string { return `manual updater` }

// Configure implements driver.Configurable.
func (u *updater) Configure(ctx context.Context, f driver.ConfigUnmarshaler, c *http.Client) error {
	ctx = zlog.ContextWithValues(ctx, "component", "updater/manual/updater.Configure")

	//client is always nil since there's no need to have http connection

	var cfg Config
	if l := len(cfg.Allowlist); l != 0 {
		u.allow = make(map[string]bool, l)
		for _, a := range cfg.Allowlist {
			u.allow[a] = true
		}
	}

	zlog.Debug(ctx).Msg("loaded incoming config")
	return nil
}

type fingerprint struct {
	Etag       string
	Ecosystems []ecoFP
}

type ecoFP struct {
	Ecosystem string
	Key       string
	Etag      string
}

func (u *updater) Fetch(ctx context.Context, f driver.Fingerprint) (io.ReadCloser, driver.Fingerprint, error) {
	return nil, "", nil
}

func (u *updater) Parse(ctx context.Context, contents io.ReadCloser) ([]*claircore.Vulnerability, error) {
	return manuallyEnrichedVulns, nil
}

type (
	fileStat interface{ Stat() (fs.FileInfo, error) }
	sizer    interface{ Size() int64 }
)

// Ecs is an entity-component system for vulnerabilities.
//
// This is organized this way to help consolidate allocations.
type ecs struct {
	Updater string

	pkgindex  map[string]int
	repoindex map[string]int

	Vulnerability []claircore.Vulnerability
	Package       []claircore.Package
	Distribution  []claircore.Distribution
	Repository    []claircore.Repository
}

func newECS(u string) ecs {
	return ecs{
		Updater:   u,
		pkgindex:  make(map[string]int),
		repoindex: make(map[string]int),
	}
}
