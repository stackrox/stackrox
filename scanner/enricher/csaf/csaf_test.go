package csaf

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/csaf"
	"github.com/stretchr/testify/assert"
)

var (
	expectedRecords = []csaf.Advisory{
		{
			Name:        "RHSA-2023:4701",
			Description: "Red Hat Security Advisory: subscription-manager security update",
			Severity:    "Moderate",
			CVSSv3: csaf.CVSS{
				Score:  6.1,
				Vector: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:H/A:L",
			},
		},
		{
			Name:        "RHSA-2023:4706",
			Description: "Red Hat Security Advisory: subscription-manager security update",
			Severity:    "Important",
			CVSSv3: csaf.CVSS{
				Score:  7.8,
				Vector: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
			},
		},
		{
			Name:        "RHSA-2024:10186",
			Description: "Red Hat Security Advisory: ACS 4.5 enhancement update",
			Severity:    "Important",
			CVSSv3: csaf.CVSS{
				Score:  8.2,
				Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:L",
			},
		},
	}

	expectedEnrichmentRecords = func() map[string]driver.EnrichmentRecord {
		expected := make(map[string]driver.EnrichmentRecord, len(expectedRecords))
		for _, r := range expectedRecords {
			b, err := json.Marshal(r)
			if err != nil {
				panic(err)
			}
			expected[r.Name] = driver.EnrichmentRecord{
				Tags:       []string{r.Name},
				Enrichment: b,
			}
		}
		return expected
	}()
)

func TestConfigure(t *testing.T) {
	t.Parallel()
	ctx := zlog.Test(context.Background(), t)

	noopConfig := func(_ interface{}) error { return nil }
	type configTestcase struct {
		Config func(interface{}) error
		Check  func(*testing.T, error)
		Name   string
	}

	tt := []configTestcase{
		{
			Name: "None",
		},
		{
			Name: "OK",
			Config: func(i interface{}) error {
				cfg := i.(*Config)
				s := "http://example.com/"
				cfg.URL = s
				return nil
			},
		},
		{
			Name:   "UnmarshalError",
			Config: func(_ interface{}) error { return errors.New("expected error") },
			Check: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected unmarshal error")
				}
			},
		},
		{
			Name: "TrailingSlashOK",
			Config: func(i interface{}) error {
				cfg := i.(*Config)
				s := "http://example.com"
				cfg.URL = s
				return nil
			},
		},
		{
			Name: "BadURL",
			Config: func(i interface{}) error {
				cfg := i.(*Config)
				s := "http://[notaurl:/"
				cfg.URL = s
				return nil
			},
			Check: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected URL parse error")
				}
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			e := &Enricher{}
			ctx := zlog.Test(ctx, t)
			f := tc.Config
			if f == nil {
				f = noopConfig
			}
			err := e.Configure(ctx, f, nil)
			if tc.Check == nil {
				if err != nil {
					t.Errorf("unexpected err: %v", err)
				}
				return
			}
			tc.Check(t, err)
		})
	}
}

type enrichmentGetter func(context.Context, []string) ([]driver.EnrichmentRecord, error)

func (eg enrichmentGetter) GetEnrichment(ctx context.Context, tags []string) ([]driver.EnrichmentRecord, error) {
	return eg(ctx, tags)
}

func TestEnrich(t *testing.T) {
	t.Parallel()
	ctx := zlog.Test(context.Background(), t)

	g := enrichmentGetter(func(ctx context.Context, tags []string) ([]driver.EnrichmentRecord, error) {
		return []driver.EnrichmentRecord{
			expectedEnrichmentRecords[tags[0]],
		}, nil
	})

	vr := &claircore.VulnerabilityReport{
		Vulnerabilities: map[string]*claircore.Vulnerability{
			"foo": {
				Name:               "CVE-2023-3899",
				Description:        "My description",
				Severity:           "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:H/A:L",
				NormalizedSeverity: claircore.Medium,
				Links:              "https://access.redhat.com/security/cve/cve-2023-3899 https://access.redhat.com/errata/RHSA-2023:4701",
				Updater:            "rhel-vex",
			},
			"bar": {
				Name:               "CVE-2023-3899",
				Description:        "My description",
				Severity:           "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:H/A:L",
				NormalizedSeverity: claircore.Medium,
				Links:              "https://access.redhat.com/security/cve/cve-2023-3899 https://access.redhat.com/errata/RHSA-2023:4706",
				Updater:            "rhel-vex",
			},
			"baz": {
				Name:               "CVE-2024-34156",
				Description:        "My description",
				Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
				NormalizedSeverity: claircore.High,
				Links:              "https://access.redhat.com/security/cve/cve-2024-34156 https://access.redhat.com/errata/RHSA-2024:10186",
				Updater:            "rhel-vex",
			},
			"not-rhel-vex": {
				Name:               "CVE-2025-21342",
				Description:        "My super description",
				Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
				NormalizedSeverity: claircore.High,
				Links:              "https://access.redhat.com/security/cve/cve-2024-34156 https://access.redhat.com/errata/RHSA-2024:10186",
				Updater:            "not-rhel-vex",
			},
			"no-rhsa": {
				Name:               "CVE-2025-21343",
				Description:        "My super duper description",
				Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
				NormalizedSeverity: claircore.High,
				Links:              "https://access.redhat.com/security/cve/cve-2025-21343",
				Updater:            "rhel-vex",
			},
		},
	}

	e := &Enricher{}
	kind, es, err := e.Enrich(ctx, g, vr)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if kind != csaf.Type {
		t.Errorf("expected kind %q, got %q", csaf.Type, kind)
	}

	var enrichments map[string][]csaf.Advisory
	err = json.Unmarshal(es[0], &enrichments)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := map[string][]csaf.Advisory{
		"foo": {expectedRecords[0]},
		"bar": {expectedRecords[1]},
		"baz": {expectedRecords[2]},
	}

	assert.Equal(t, expected, enrichments)
}
