package nvd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/facebookincubator/nvdtools/cveapi/nvd/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/nvd"
)

func TestConfigure(t *testing.T) {
	t.Parallel()
	ctx := zlog.Test(context.Background(), t)
	tt := []configTestcase{
		{
			Name: "None",
		},
		{
			Name: "OK",
			Config: func(i interface{}) error {
				cfg := i.(*Config)
				s := "http://example.com/"
				cfg.FeedRoot = &s
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
			Name: "TrailingSlash",
			Config: func(i interface{}) error {
				cfg := i.(*Config)
				s := "http://example.com"
				cfg.FeedRoot = &s
				return nil
			},
			Check: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected trailing slash error")
				}
			},
		},
		{
			Name: "BadURL",
			Config: func(i interface{}) error {
				cfg := i.(*Config)
				s := "http://[notaurl:/"
				cfg.FeedRoot = &s
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
		t.Run(tc.Name, tc.Run(ctx))
	}
}

type configTestcase struct {
	Config func(interface{}) error
	Check  func(*testing.T, error)
	Name   string
}

func (tc configTestcase) Run(ctx context.Context) func(*testing.T) {
	e := &Enricher{}
	return func(t *testing.T) {
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
	}
}

func noopConfig(_ interface{}) error { return nil }

func TestFetch(t *testing.T) {
	t.Parallel()
	ctx := zlog.Test(context.Background(), t)
	srv := mockServer(t)
	tt := []fetchTestcase{
		{
			Name: "Happy Case",
			Check: func(t *testing.T, rc io.ReadCloser, fp driver.Fingerprint, err error) {
				if rc == nil {
					t.Error("wanted non-nil ReadCloser")
				}
				t.Logf("got error: %v", err)
				if err != nil {
					t.Error("wanted nil error")
				}
			},
		},
		{
			Name:     "With NVD ZIP",
			FeedPath: filepath.Join("testdata", "feed.zip"),
			Check: func(t *testing.T, rc io.ReadCloser, fp driver.Fingerprint, err error) {
				if rc == nil {
					t.Fatalf("wanted non-nil ReadCloser")
				}
				t.Logf("got error: %v", err)
				if err != nil {
					t.Fatalf("wanted nil error")
				}
				enrichments := make(map[string]driver.EnrichmentRecord)
				dec := json.NewDecoder(rc)
				for {
					var e driver.EnrichmentRecord
					if err = dec.Decode(&e); err != nil {
						break
					}
					enrichments[e.Tags[0]] = e
				}
				if !errors.Is(err, io.EOF) {
					t.Fatalf("wanted EOF, found: %v", err)
				}
				// Look for some CVEs.
				_, ok := enrichments["CVE-2023-50612"]
				if !ok {
					t.Fatal("CVE-2023-50612 not found")
				}
				_, ok = enrichments["CVE-2023-50609"]
				if !ok {
					t.Fatal("CVE-2023-50609 not found")
				}
				enrichment, ok := enrichments["CVE-2017-18349"]
				if !ok {
					t.Fatal("CVE-2017-18349 not found")
				}
				var item schema.CVEAPIJSON20CVEItem
				if err := json.Unmarshal(enrichment.Enrichment, &item); err != nil {
					t.Fatalf("could not unmarshal CVE-2017-18349 enrichment: %v", err)
				}
				if item.Metrics == nil || item.Metrics.CvssMetricV31 != nil {
					t.Fatal("unexpected values for CVE-2017-18349")
				}
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, tc.Run(ctx, srv))
	}
}

type fetchTestcase struct {
	Check    func(*testing.T, io.ReadCloser, driver.Fingerprint, error)
	Name     string
	Hint     string
	FeedPath string
}

func (tc fetchTestcase) Run(ctx context.Context, srv *httptest.Server) func(*testing.T) {
	e := &Enricher{}
	return func(t *testing.T) {
		ctx := zlog.Test(ctx, t)
		f := func(i interface{}) error {
			cfg, ok := i.(*Config)
			if !ok {
				t.Fatal("assertion failed")
			}
			u := srv.URL + "/"
			cfg.FeedRoot = &u
			ci := "0"
			// Speed up the test.
			cfg.CallInterval = &ci
			if tc.FeedPath != "" {
				cfg.FeedPath = &tc.FeedPath
			}
			return nil
		}
		if err := e.Configure(ctx, f, srv.Client()); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		rc, fp, err := e.FetchEnrichment(ctx, driver.Fingerprint(tc.Hint))
		if rc != nil {
			defer func() {
				_ = rc.Close()
			}()
		}
		if tc.Check == nil {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			return
		}
		tc.Check(t, rc, fp, err)
	}
}

func mockServer(t *testing.T) *httptest.Server {
	const root = `testdata/`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return only the first page in the testdata, otherwise all empty.
		if !strings.HasPrefix(r.FormValue("pubStartDate"), "2002-") || r.FormValue("startIndex") != "0" {
			_, _ = fmt.Fprint(w, `{"resultsPerPage": 0, "totalResults": 0}`)
			return
		}
		f, err := os.Open(filepath.Join(root, "feed.json"))
		if err != nil {
			t.Errorf("open failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer func() {
			_ = f.Close()
		}()
		if _, err := io.Copy(w, f); err != nil {
			t.Errorf("write error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestParse(t *testing.T) {
	t.Parallel()
	ctx := zlog.Test(context.Background(), t)
	srv := mockServer(t)
	tt := []parseTestcase{
		{
			Name: "OK",
		},
	}
	for _, tc := range tt {
		t.Run(tc.Name, tc.Run(ctx, srv))
	}
}

type parseTestcase struct {
	Check func(*testing.T, []driver.EnrichmentRecord, error)
	Name  string
}

func (tc parseTestcase) Run(ctx context.Context, srv *httptest.Server) func(*testing.T) {
	e := &Enricher{}
	return func(t *testing.T) {
		ctx := zlog.Test(ctx, t)
		f := func(i interface{}) error {
			cfg, ok := i.(*Config)
			if !ok {
				t.Fatal("assertion failed")
			}
			u := srv.URL + "/"
			cfg.FeedRoot = &u
			// Speed up the test.
			ci := "0"
			cfg.CallInterval = &ci
			return nil
		}
		if err := e.Configure(ctx, f, srv.Client()); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		rc, _, err := e.FetchEnrichment(ctx, "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		defer func() {
			_ = rc.Close()
		}()
		rs, err := e.ParseEnrichment(ctx, rc)
		if tc.Check == nil {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			return
		}
		tc.Check(t, rs, err)
	}
}

func TestEnrich(t *testing.T) {
	t.Parallel()
	ctx := zlog.Test(context.Background(), t)
	g := newFakeGetter(t, "testdata/feed.json")
	r := &claircore.VulnerabilityReport{
		Vulnerabilities: map[string]*claircore.Vulnerability{
			"-1": {
				Description: "This is a fake vulnerability that doesn't have a CVE.",
			},
			"1": {
				Description: "This is a fake vulnerability that looks like CVE-2023-50612.",
			},
			"6004": {
				Description: "CVE-2020-6004 was unassigned",
			},
			"6005": {
				Description: "CVE-2023-50612 duplicate",
			},
		},
	}
	e := &Enricher{}
	kind, es, err := e.Enrich(ctx, g, r)
	if err != nil {
		t.Error(err)
	}
	if got, want := kind, nvd.Type; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
	want := map[string][]map[string]interface{}{
		"1": {{
			"descriptions": []any{
				map[string]any{"lang": string("en"), "value": "Insecure Permissions vulnerability in fit2cloud Cloud Explorer Lite version 1.4.1, allow local attackers to escalate privileges and obtain sensitive information via the cloud accounts parameter."},
			},
			"id":           "CVE-2023-50612",
			"lastModified": "2024-01-11T15:02:43.727",
			"published":    "2024-01-06T03:15:43.990",
			"metrics": map[string]any{
				"cvssMetricV31": []any{
					map[string]any{
						"cvssData": map[string]any{
							"baseScore":    float64(7.8),
							"baseSeverity": "",
							"vectorString": "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
							"version":      "3.1",
						},
						"source": "",
						"type":   "",
					}}},
			"references": nil,
		}},
		"6005": {{
			"descriptions": []any{
				map[string]any{"lang": string("en"), "value": "Insecure Permissions vulnerability in fit2cloud Cloud Explorer Lite version 1.4.1, allow local attackers to escalate privileges and obtain sensitive information via the cloud accounts parameter."},
			},
			"id":           "CVE-2023-50612",
			"lastModified": "2024-01-11T15:02:43.727",
			"published":    "2024-01-06T03:15:43.990",
			"metrics": map[string]any{
				"cvssMetricV31": []any{
					map[string]any{
						"cvssData": map[string]any{
							"baseScore":    float64(7.8),
							"baseSeverity": "",
							"vectorString": "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
							"version":      "3.1",
						},
						"source": "",
						"type":   "",
					}}},
			"references": nil,
		}},
	}
	got := map[string][]map[string]interface{}{}
	if err := json.Unmarshal(es[0], &got); err != nil {
		t.Error(err)
	}
	if !cmp.Equal(got, want) {
		t.Error(cmp.Diff(got, want))
	}
}

func newFakeGetter(t *testing.T, path string) driver.EnrichmentGetter {
	feedIn, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = feedIn.Close()
	}()
	var data *schema.CVEAPIJSON20
	err = json.NewDecoder(feedIn).Decode(&data)
	if err != nil {
		t.Fatal(err)
	}
	g := &fakeGetter{items: map[string]json.RawMessage{}}
	for _, v := range data.Vulnerabilities {
		en, err := json.Marshal(filterFields(v.CVE))
		if err != nil {
			t.Fatal(err)
		}
		g.items[v.CVE.ID] = en
	}
	return g
}

type fakeGetter struct {
	res   []driver.EnrichmentRecord
	items map[string]json.RawMessage
}

func (f *fakeGetter) GetEnrichment(_ context.Context, tags []string) ([]driver.EnrichmentRecord, error) {
	id := tags[0]
	if e, ok := f.items[id]; ok {
		r := []driver.EnrichmentRecord{
			{Tags: tags, Enrichment: e},
		}
		f.res = r
		return r, nil
	}
	return nil, nil
}
