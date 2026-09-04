package csaf

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/klauspost/compress/snappy"
	"github.com/quay/claircore/test"
	testvex "github.com/quay/claircore/test/vex"
	"github.com/quay/claircore/toolkit/types/csaf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchEnrichment(t *testing.T) {
	ctx := test.Logging(t)
	root, c := testvex.ServeSecDB(ctx, t, "testdata/server.txtar")
	enricher := &Enricher{}
	err := enricher.Configure(ctx, func(v interface{}) error {
		cf := v.(*Config)
		cf.URL = root + "/"
		return nil
	}, c)
	if err != nil {
		t.Fatal(err)
	}

	data, fp, err := enricher.FetchEnrichment(ctx, "")
	if err != nil {
		t.Fatalf("error Fetching, cannot continue: %v", err)
	}
	t.Cleanup(func() {
		if err := data.Close(); err != nil {
			t.Errorf("error closing data: %v", err)
		}
	})
	// Check fingerprint.
	f, err := parseFingerprint(fp)
	if err != nil {
		t.Errorf("fingerprint cannot be parsed: %v", err)
	}
	if f.changesEtag != "something" {
		t.Errorf("bad etag for the changes.csv endpoint: %s", f.changesEtag)
	}

	// Check saved vulns
	expectedLnCt := 2
	lnCt := 0
	r := bufio.NewReader(snappy.NewReader(data))
	for b, err := r.ReadBytes('\n'); err == nil; b, err = r.ReadBytes('\n') {
		_, err := csaf.Parse(bytes.NewReader(b))
		if err != nil {
			t.Error(err)
		}
		lnCt++
	}
	if lnCt != expectedLnCt {
		t.Errorf("got %d entries but expected %d", lnCt, expectedLnCt)
	}
}

func TestFetchAdvisory_RetriesOnFailure(t *testing.T) {
	origRetries := advisoryFetchRetries
	origBaseDelay := advisoryFetchRetryBaseDelay
	advisoryFetchRetries = 3
	advisoryFetchRetryBaseDelay = 1 * time.Millisecond
	t.Cleanup(func() {
		advisoryFetchRetries = origRetries
		advisoryFetchRetryBaseDelay = origBaseDelay
	})

	validJSON := `{"document":{"tracking":{"id":"RHSA-2024:0001"}}}`

	t.Run("succeeds after transient failure", func(t *testing.T) {
		var attempts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if attempts.Add(1) < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fmt.Fprint(w, validJSON)
		}))
		t.Cleanup(srv.Close)

		e := &Enricher{c: srv.Client()}
		u, _ := url.Parse(srv.URL + "/advisory.json")
		var buf, bc bytes.Buffer
		var w bytes.Buffer
		err := e.fetchAdvisory(context.Background(), u, &buf, &bc, &w)
		require.NoError(t, err)
		assert.Equal(t, int32(3), attempts.Load())
		assert.Contains(t, w.String(), "RHSA-2024:0001")
	})

	t.Run("returns error after all retries exhausted", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "not found")
		}))
		t.Cleanup(srv.Close)

		e := &Enricher{c: srv.Client()}
		u, _ := url.Parse(srv.URL + "/advisory.json")
		var buf, bc bytes.Buffer
		var w bytes.Buffer
		err := e.fetchAdvisory(context.Background(), u, &buf, &bc, &w)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
		assert.Empty(t, w.String())
	})
}

func TestProcessChanges_SkipsUnavailableAdvisory(t *testing.T) {
	origRetries := advisoryFetchRetries
	advisoryFetchRetries = 1
	t.Cleanup(func() {
		advisoryFetchRetries = origRetries
	})

	validJSON := `{"document":{"tracking":{"id":"RHSA-2024:0001"}}}`
	mux := http.NewServeMux()
	mux.HandleFunc("/changes.csv", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Etag", "test-etag")
		// Two advisories: one available, one returning 404.
		fmt.Fprint(w,
			`"2024/rhsa-2024_0001.json","2025-01-10T18:37:32+00:00"`+"\n"+
				`"2024/rhsa-2024_0002.json","2025-01-10T18:37:32+00:00"`)
	})
	mux.HandleFunc("/2024/rhsa-2024_0001.json", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, validJSON)
	})
	mux.HandleFunc("/2024/rhsa-2024_0002.json", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "<!DOCTYPE html><html>not found</html>")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	u, _ := url.Parse(srv.URL + "/")
	e := &Enricher{c: srv.Client(), base: u}
	fp := &fingerprint{}
	changed := map[string]bool{}
	var w bytes.Buffer

	err := e.processChanges(context.Background(), &w, fp, changed)
	require.NoError(t, err)
	assert.True(t, changed["rhsa-2024_0001.json"], "available advisory should be marked as changed")
	assert.False(t, changed["rhsa-2024_0002.json"], "404'd advisory should not be marked as changed")
	assert.Contains(t, w.String(), "RHSA-2024:0001")
	assert.NotContains(t, w.String(), "RHSA-2024:0002")
}
