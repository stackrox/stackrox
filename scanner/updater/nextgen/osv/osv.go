package osv

import (
	"archive/zip"
	"context"
	"io/fs"
	"net/http"

	libvuln "github.com/quay/claircore/libvuln/driver"
	v1 "github.com/quay/claircore/updater/driver/v1"
	libvulnOsv "github.com/quay/claircore/updater/osv"
)

// Config defines the configuration for osv Updater
type Config struct {
	URL       string   `json:"url" yaml:"url"`
	Allowlist []string `json:"allowlist" yaml:"allowlist"`
}

// UpdaterFactory implements interfaces.UpdaterFactory
type UpdaterFactory struct{}

func (f *UpdaterFactory) Name() string {
	return "osv"
}

func (f *UpdaterFactory) Create(ctx context.Context, unmarshaler v1.ConfigUnmarshaler) ([]v1.Updater, error) {
	var osvCfg Config
	err := unmarshaler(&osvCfg)
	if err != nil {
		return nil, err
	}

	libVulnFac := &libvulnOsv.Factory{}
	client := &http.Client{}
	libVulnCfg := func(v interface{}) error {
		facCfg := v.(*libvulnOsv.FactoryConfig)
		facCfg.URL = osvCfg.URL
		facCfg.Allowlist = osvCfg.Allowlist
		return nil
	}

	if err := libVulnFac.Configure(ctx, libVulnCfg, client); err != nil {
		return nil, err
	}

	libVulnUpdSet, err := libVulnFac.UpdaterSet(ctx)
	if err != nil {
		return nil, err
	}

	var updaters []v1.Updater
	for _, libVulnUpd := range libVulnUpdSet.Updaters() {
		wrappedUpdater := &Updater{
			libvulnUpdater: libVulnUpd,
		}
		updaters = append(updaters, wrappedUpdater)
	}

	return updaters, nil
}

// Updater implements v1.Updater
type Updater struct {
	libvulnUpdater libvuln.Updater
}

func (u *Updater) Name() string {
	return u.libvulnUpdater.Name()
}

func (u *Updater) Fetch(ctx context.Context, writer *zip.Writer, fingerprint v1.Fingerprint, client *http.Client) (v1.Fingerprint, error) {
	return "", nil
}

func (u *Updater) ParseVulnerability(ctx context.Context, data fs.FS) (*v1.ParsedVulnerabilities, error) {

	parsedVulns := &v1.ParsedVulnerabilities{
		// ... Populate ParsedVulnerabilities object
	}

	return parsedVulns, nil
}

// ParseEnrichment implements interfaces.EnrichmentParser
func (u *Updater) ParseEnrichment(ctx context.Context, data fs.FS) ([]v1.EnrichmentRecord, error) {
	return nil, nil
}
