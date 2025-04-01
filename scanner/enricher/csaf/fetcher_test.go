package csaf

import (
	"bufio"
	"bytes"
	"context"
	"testing"

	"github.com/klauspost/compress/snappy"
	"github.com/quay/claircore/rhel/vex"
	"github.com/quay/claircore/toolkit/types/csaf"
	"github.com/quay/zlog"
)

func TestFetchEnrichment(t *testing.T) {
	ctx := zlog.Test(context.Background(), t)
	root, c := vex.ServeSecDB(t, "testdata/server.txtar")
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
