// Package manual provides a custom updater for vulnerability scanners.
// This updater allows manual input of vulnerability data.
package manual

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
)

const Name = "stackrox-manual"
const DefaultURL = "https://raw.githubusercontent.com/stackrox/stackrox/master/scanner/updater/manual/vulns.json"

type updater struct {
	c   *http.Client
	uri *url.URL
}

// NewUpdater creates a new instance of the updater with default settings.
func NewUpdater(c *http.Client, uri string) (*updater, error) {
	var u *url.URL
	var err error
	if uri == "" {
		u, err = url.Parse(DefaultURL)
	}
	u, err = url.Parse(uri)
	if err != nil {
		return nil, err
	}
	return &updater{
		c:   c,
		uri: u,
	}, nil
}

// Configure allows configuration of the updater, including changing the source URL.
func (u *updater) Configure(ctx context.Context, f driver.ConfigUnmarshaler, c *http.Client) error {
	ctx = zlog.ContextWithValues(ctx, "component", "updater/stackrox-manual/updater.Configure")
	var err error

	u.c = c
	var cfg UpdaterConfig
	if err := f(&cfg); err != nil {
		return err
	}
	if cfg.URI != "" {
		u.uri, err = url.Parse(cfg.URI)
		if err != nil {
			return err
		}
	}
	zlog.Debug(ctx).Msg("loaded incoming config")
	return nil
}

type UpdaterConfig struct {
	Client *http.Client
	URI    string
}

// Name returns the name of the updater.
func (u *updater) Name() string {
	return Name
}

// Fetch fetching data from a configurable URI.
func (u *updater) Fetch(ctx context.Context, fingerprint driver.Fingerprint) (io.ReadCloser, driver.Fingerprint, error) {
	resp, err := u.c.Get(u.uri.String())
	if err != nil {
		return nil, "", err
	}
	return resp.Body, "", nil
}

// Parse parsing the fetched json file into vulnerabilities.
func (u *updater) Parse(ctx context.Context, rc io.ReadCloser) ([]*claircore.Vulnerability, error) {
	// Actual parsing logic goes here.
	// This is a stub, replace with actual data handling logic.
	return []*claircore.Vulnerability{}, nil
}

// UpdaterSet initializes an updater set with a configured updater based on provided URI and client.
func UpdaterSet(ctx context.Context, client *http.Client, uri string) (driver.UpdaterSet, error) {
	// Create a new updater instance with the provided client.
	updater, err := NewUpdater(client)
	if err != nil {
		return nil, err
	}

	// Configure the updater with the provided URI.
	if uri == "" {
		uri = DefaultURL
	}
	// Manually set the URI within the updater's configuration.
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}
	updater.uri = parsedURI // Directly setting the URI to the updater

	// Create a new updater set and add the configured updater.
	updaterSet := driver.NewUpdaterSet()
	if err := updaterSet.Add(updater); err != nil {
		return nil, fmt.Errorf("failed to add updater to set: %w", err)
	}

	return updaterSet, nil
}
