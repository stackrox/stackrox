package notaffected

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/klauspost/compress/snappy"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/toolkit/types/csaf"
	"github.com/quay/zlog"
)

func TestFactory(t *testing.T) {
	ctx := zlog.Test(t.Context(), t)
	root, c := ServeSecDB(t, "testdata/server.txtar")
	e := &Enricher{}
	err := e.Configure(ctx, func(v interface{}) error {
		cf := v.(*Config)
		cf.URL = root + "/"
		return nil
	}, c)
	if err != nil {
		t.Fatal(err)
	}

	data, fp, err := e.FetchEnrichment(ctx, "")
	if err != nil {
		t.Fatalf("error Fetching, cannot continue: %v", err)
	}
	defer func() {
		_ = data.Close()
	}()
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

	newData, newFP, err := e.FetchEnrichment(ctx, driver.Fingerprint(f.String()))
	if err != nil {
		t.Fatalf("error re-Fetching, cannot continue: %v", err)
	}
	defer func() {
		_ = newData.Close()
	}()

	f, err = parseFingerprint(newFP)
	if err != nil {
		t.Errorf("fingerprint cannot be parsed: %v", err)
	}
	if f.changesEtag != "something" {
		t.Errorf("bad etag for the changes.csv endpoint: %s", f.changesEtag)
	}

	r = bufio.NewReader(snappy.NewReader(newData))
	for _, err := r.ReadBytes('\n'); err == nil; _, err = r.ReadBytes('\n') {
		t.Fatal("should not have anymore data")
	}
}
