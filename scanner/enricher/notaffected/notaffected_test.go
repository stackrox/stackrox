package notaffected

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/csaf"
	pkgnotaffected "github.com/stackrox/rox/pkg/scannerv4/enricher/notaffected"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestConfigure(t *testing.T) {
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

func enrichRecords(records map[string][]string) map[string]driver.EnrichmentRecord {
	enriched := make(map[string]driver.EnrichmentRecord, len(records))
	for k, v := range records {
		b, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		enriched[k] = driver.EnrichmentRecord{
			Tags:       []string{k},
			Enrichment: b,
		}
	}
	return enriched
}

func TestEnrich(t *testing.T) {
	testutils.MustUpdateFeature(t, features.ScannerV4KnownNotAffected, true)

	ctx := zlog.Test(t.Context(), t)

	expectedRecords := map[string][]string{
		"openshift4/ose-kube-rbac-proxy-rhel9": {"CVE-2024-45337"},
		"red_hat_products":                     {"CVE-2024-21613"},
	}
	enrichedRecords := enrichRecords(expectedRecords)

	g := enrichmentGetter(func(_ context.Context, tags []string) ([]driver.EnrichmentRecord, error) {
		return []driver.EnrichmentRecord{enrichedRecords[tags[0]]}, nil
	})

	vr := &claircore.VulnerabilityReport{
		Packages: map[string]*claircore.Package{
			"910031": {
				ID:      "910031",
				Name:    "openshift4/ose-kube-rbac-proxy-rhel9",
				Version: "v4.18.0-202505200035.p0.g526498a.assembly.stream.el9",
				Kind:    claircore.BINARY,
			},
			"910165": {
				ID:      "910165",
				Name:    "golang.org/x/crypto",
				Version: "v0.26.0",
				Kind:    claircore.BINARY,
			},
		},
		Repositories: map[string]*claircore.Repository{
			"1": {
				ID:   "1",
				Name: "Red Hat Container Catalog",
				URI:  "https://catalog.redhat.com/software/containers/explore",
				CPE:  cpe.MustUnbind("cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"),
			},
			"2": {
				ID:   "2",
				Name: "cpe:/o:redhat:rhel_eus:9.4::baseos",
				Key:  "rhel-cpe-repository",
				CPE:  cpe.MustUnbind("cpe:2.3:o:redhat:rhel_eus:9.4:*:baseos:*:*:*:*:*"),
			},
			"3": {
				ID:   "3",
				Name: "cpe:/a:redhat:rhel_eus:9.4::appstream",
				Key:  "rhel-cpe-repository",
				CPE:  cpe.MustUnbind("cpe:2.3:a:redhat:rhel_eus:9.4:*:appstream:*:*:*:*:*"),
			},
			"10": {
				ID:   "10",
				Name: "go",
				URI:  "https://pkg.go.dev/",
				CPE:  cpe.MustUnbind("cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"),
			},
			"3621": {
				ID:   "3621",
				Name: "cpe:/a:redhat:openshift:4.18::el9",
				Key:  "rhel-cpe-repository",
				CPE:  cpe.MustUnbind("cpe:2.3:a:redhat:openshift:4.18:*:el9:*:*:*:*:*"),
			},
		},
		Environments: map[string][]*claircore.Environment{
			"910031": {
				{
					PackageDB:     "root/buildinfo/Dockerfile-openshift-ose-kube-rbac-proxy-rhel9-v4.18.0-202505200035.p0.g526498a.assembly.stream.el9",
					IntroducedIn:  claircore.MustParseDigest("sha256:0cdc5a5066ab7b763c7be38cd2a8754d75dee2e2070cf71003fed9a9be5c2747"),
					RepositoryIDs: []string{"1"},
				},
			},
			"910165": {
				{
					PackageDB:     "go:usr/bin/kube-rbac-proxy",
					IntroducedIn:  claircore.MustParseDigest("sha256:0cdc5a5066ab7b763c7be38cd2a8754d75dee2e2070cf71003fed9a9be5c2747"),
					RepositoryIDs: []string{"10"},
				},
			},
		},
		Vulnerabilities: map[string]*claircore.Vulnerability{
			"327505": {
				ID:                 "327505",
				Name:               "CVE-2024-45337",
				Description:        "Misuse of ServerConfig.PublicKeyCallback may cause authorization bypass in golang.org/x/crypto",
				Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N",
				NormalizedSeverity: claircore.High,
				Links:              "idk",
				Updater:            "osv/go",
			},
		},
	}

	e := &Enricher{}
	kind, es, err := e.Enrich(ctx, g, vr)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if kind != pkgnotaffected.Type {
		t.Errorf("expected kind %q, got %q", csaf.Type, kind)
	}

	var enrichments map[string][][]string
	err = json.Unmarshal(es[0], &enrichments)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := map[string][][]string{
		"openshift4/ose-kube-rbac-proxy-rhel9": {{"CVE-2024-45337"}},
		"red_hat_products":                     {{"CVE-2024-21613"}},
	}

	assert.Equal(t, expected, enrichments)
}
