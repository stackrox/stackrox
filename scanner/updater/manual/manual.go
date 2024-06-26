// Package manual provides a custom updater for vulnerability scanners.
// This updater allows manual input of vulnerability data.
package manual

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/utils"
	yaml "gopkg.in/yaml.v3"
)

type Vulnerability struct {
	Name               string `yaml:"Name"`
	Description        string `yaml:"Description"`
	Issued             string `yaml:"Issued"`
	Links              string `yaml:"Links"`
	Severity           string `yaml:"Severity"`
	NormalizedSeverity string `yaml:"NormalizedSeverity"`
	Package            struct {
		Name           string `yaml:"Name"`
		Kind           string `yaml:"Kind"`
		RepositoryHint string `yaml:"RepositoryHint"`
	} `yaml:"Package"`
	FixedInVersion string `yaml:"FixedInVersion"`
	Repo           struct {
		Name string `yaml:"Name"`
		URI  string `yaml:"URI"`
	} `yaml:"Repo"`
}

const (
	Name = "stackrox-manual"
	// DefaultURL Default URL to fetch the vulnerabilities JSON.
	DefaultURL = "https://raw.githubusercontent.com/stackrox/stackrox/master/scanner/updater/manual/vulns.yaml"
)

var client = &http.Client{
	Timeout: 5 * time.Minute,
}

type updater struct {
	c         *http.Client
	updateURL *url.URL
}

// NewUpdater creates a new instance of the updater with default settings.
func NewUpdater(c *http.Client, uri string) (*updater, error) {
	var url *url.URL
	var err error
	if uri == "" {
		uri = DefaultURL
	}
	url, err = url.Parse(uri)
	if err != nil {
		return nil, err
	}
	return &updater{c: c, updateURL: url}, nil
}

func (u *updater) Name() string {
	return Name
}

// Fetch fetching data from a configurable URI.
func (u *updater) Fetch(ctx context.Context, fingerprint driver.Fingerprint) (io.ReadCloser, driver.Fingerprint, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "updater/manual/manual.Fetch")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.updateURL.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := u.c.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch data: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			zlog.Error(ctx).Err(err).Msg("failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("bad response status: %s", resp.Status)
	}
	out, err := os.CreateTemp("", "manual")
	if err != nil {
		return nil, "", errors.Wrap(err, "creating scanner defs file")
	}
	zlog.Debug(ctx).
		Str("filename", out.Name()).
		Msg("opened temporary file for output")

	utils.IgnoreError(func() error {
		return os.RemoveAll(out.Name())
	})
	_, err = io.Copy(out, resp.Body)

	if err != nil {
		utils.IgnoreError(out.Close)
		return nil, "", fmt.Errorf("failed to write to temporary file: %w", err)
	}

	if _, err = out.Seek(0, io.SeekStart); err != nil {
		utils.IgnoreError(out.Close)
		return nil, "", fmt.Errorf("seek failed: %w", err)
	}
	zlog.Info(ctx).
		Str("dir", out.Name()).
		Msg("fetched manual vulnerability yaml file")
	return out, "", nil
}

// Parse parsing the fetched yaml file into vulnerabilities.
func (u *updater) Parse(ctx context.Context, rc io.ReadCloser) ([]*claircore.Vulnerability, error) {
	defer func() {
		_ = rc.Close()
	}()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var vulnerabilities struct {
		Vulnerabilities []Vulnerability `yaml:"vulnerabilities"`
	}

	if err := yaml.Unmarshal(data, &vulnerabilities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	var clairVulns []*claircore.Vulnerability
	for _, v := range vulnerabilities.Vulnerabilities {
		parsedTime, err := time.Parse(time.RFC3339, v.Issued)
		if err != nil {
			return nil, err
		}
		cv := &claircore.Vulnerability{
			Updater:            u.Name(),
			Name:               v.Name,
			Description:        v.Description,
			Issued:             parsedTime,
			Links:              v.Links,
			Severity:           v.Severity,
			NormalizedSeverity: severity(v.NormalizedSeverity),
			Package: &claircore.Package{
				Name:           v.Package.Name,
				Kind:           claircore.BINARY,
				RepositoryHint: v.Package.RepositoryHint,
			},
			FixedInVersion: v.FixedInVersion,
			Repo: &claircore.Repository{
				Name: v.Repo.Name,
				URI:  v.Repo.URI,
			},
		}
		clairVulns = append(clairVulns, cv)
	}

	zlog.Info(ctx).
		Int("count", len(clairVulns)).
		Msg("All manual vulnerabilities parsed")
	return clairVulns, nil
}

// UpdaterSet initializes an updater set with a configured updater based on provided URI and client.
func UpdaterSet(ctx context.Context, uri string) (driver.UpdaterSet, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "updater/manual/manual.UpdaterSet")
	res := driver.NewUpdaterSet()
	u, err := NewUpdater(client, uri)
	if err != nil {
		return res, err
	}

	if err := res.Add(u); err != nil {
		return res, fmt.Errorf("failed to create new updater set: %w", err)
	}
	zlog.Info(ctx).
		Str("url", u.updateURL.String()).
		Msg("created manual updater set")
	return res, nil
}
