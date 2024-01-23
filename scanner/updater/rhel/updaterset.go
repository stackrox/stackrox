package rhel

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/updater/rhel/internal/pulp"
)

// DefaultManifest is the url for the Red Hat OVAL pulp repository.
//
//doc:url updater
const DefaultManifest = `https://access.redhat.com/security/data/oval/v2/PULP_MANIFEST`

// NewFactory creates a Factory making updaters based on the contents of the
// provided pulp manifest.
func NewFactory(ctx context.Context, manifest string) (*Factory, error) {
	var err error
	var f Factory
	f.url, err = url.Parse(manifest)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

// Factory contains the configuration for fetching and parsing a Pulp manifest.
type Factory struct {
	url             *url.URL
	client          *http.Client
	manifestEtag    string
	ignoreUnpatched bool
}

// FactoryConfig is the configuration accepted by the rhel updaters.
//
// By convention, this should be in a map called "rhel".
type FactoryConfig struct {
	URL string `json:"url" yaml:"url"`
	// IgnoreUnpatched dictates whether to ingest unpatched advisory data
	// from the RHEL security feeds.
	IgnoreUnpatched bool `json:"ignore_unpatched" yaml:"ignore_unpatched"`
}

var _ driver.Configurable = (*Factory)(nil)

// Configure implements [driver.Configurable].
func (f *Factory) Configure(ctx context.Context, cfg driver.ConfigUnmarshaler, c *http.Client) error {
	ctx = zlog.ContextWithValues(ctx, "component", "rhel/Factory.Configure")
	var fc FactoryConfig

	if err := cfg(&fc); err != nil {
		return err
	}
	zlog.Debug(ctx).Msg("loaded incoming config")

	if fc.URL != "" {
		u, err := url.Parse(fc.URL)
		if err != nil {
			return err
		}
		zlog.Info(ctx).
			Stringer("url", u).
			Msg("configured manifest URL")
		f.url = u
	}

	if c != nil {
		zlog.Info(ctx).
			Msg("configured HTTP client")
		f.client = c
	}
	f.ignoreUnpatched = fc.IgnoreUnpatched
	return nil
}

// UpdaterSet implements [driver.UpdaterSetFactory].
//
// The returned Updaters determine the [claircore.Distribution] it's associated
// with based on the path in the Pulp manifest.
func (f *Factory) UpdaterSet(ctx context.Context) (driver.UpdaterSet, error) {
	s := driver.NewUpdaterSet()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url.String(), nil)
	if err != nil {
		return s, err
	}
	if f.manifestEtag != "" {
		req.Header.Set("if-none-match", f.manifestEtag)
	}

	res, err := f.client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return s, err
	}

	switch res.StatusCode {
	case http.StatusOK:
		if t := f.manifestEtag; t == "" || t != res.Header.Get("etag") {
			break
		}
		fallthrough
	case http.StatusNotModified:
		// return stub updater to allow us to record that all rhel updaters are up to date
		stubUpdater := Updater{name: "rhel-all"}
		s.Add(&stubUpdater)
		return s, nil
	default:
		return s, fmt.Errorf("unexpected response: %v", res.Status)
	}

	m := pulp.Manifest{}
	if err := m.Load(res.Body); err != nil {
		return s, err
	}

	for _, e := range m {
		name := strings.TrimSuffix(strings.Replace(e.Path, "/", "-", -1), ".oval.xml.bz2")
		// We need to disregard this OVAL stream because some advisories therein have
		// been released with the CPEs identical to those used in classic RHEL stream.
		// This in turn causes false CVEs to appear in scanned images. Red Hat Product
		// Security is working on fixing this situation and the plan is to remove this
		// exception in the future.
		if name == "RHEL7-rhel-7-alt" {
			continue
		}
		uri, err := f.url.Parse(e.Path)
		if err != nil {
			return s, err
		}
		m := guessFromPath.FindStringSubmatch(uri.Path)
		if m == nil {
			continue
		}
		r, err := strconv.Atoi(m[1])
		if err != nil {
			zlog.Info(ctx).
				Err(err).
				Str("path", uri.Path).
				Msg("unable to parse pattern into int")
			continue
		}
		up, err := NewUpdater(name, r, uri.String(), f.ignoreUnpatched)
		if err != nil {
			return s, err
		}
		_ = s.Add(up)
	}
	f.manifestEtag = res.Header.Get("etag")

	return s, nil
}

// Updaters returns a list of pre-configured RHEL updaters. It configures the
// factory, and returns the updaters from the factored updaterset.
func Updaters(ctx context.Context, c *http.Client) ([]driver.Updater, error) {
	f, err := NewFactory(ctx, DefaultManifest)
	if err != nil {
		return nil, err
	}
	nilCfg := func(any) error { return nil }
	f.Configure(ctx, nilCfg, c)
	s, err := f.UpdaterSet(ctx)
	if err != nil {
		return nil, err
	}
	return s.Updaters(), nil
}

var guessFromPath = regexp.MustCompile(`RHEL([0-9]+)`)
