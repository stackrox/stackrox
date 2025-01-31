package zreader

import (
	"bytes"
	"encoding/hex"
	"io"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
)

// There's no encoder in the stdlib or github.com/klauspost/compress for bzip2, so no testing for it.
//
// It should be fine, as it's using all the same logic but with a different byte string.

var testcases = []struct {
	Kind   Compression
	Values func([]reflect.Value, *rand.Rand)
}{
	{KindNone, blobUncompressed},
	{KindGzip, blobGzip},
	{KindZstd, blobZstd},
}

func TestDetect(t *testing.T) {
	t.Parallel()

	for _, tc := range testcases {
		t.Run(tc.Kind.String(), func(t *testing.T) {
			wantKind := func(k Compression) func(_, _ *bytes.Buffer) bool {
				return func(_, z *bytes.Buffer) bool {
					_, d, err := Detect(bytes.NewReader(z.Bytes()))
					if err != nil {
						t.Fatal(err)
					}
					ok := d == k
					if !ok {
						t.Errorf("guessed detection incorrectly! got: %v, want: %v", d, k)
					}
					return ok
				}
			}
			if err := quick.Check(wantKind(tc.Kind), &quick.Config{Values: tc.Values}); err != nil {
				asCheckError(t, err)
			}
		})
	}
}

func TestRead(t *testing.T) {
	t.Parallel()

	for _, tc := range testcases {
		t.Run(tc.Kind.String(), func(t *testing.T) {
			check := func(k Compression) func(_, _ *bytes.Buffer) bool {
				return func(want, z *bytes.Buffer) bool {
					r, err := Reader(z)
					if err != nil {
						t.Fatal(err)
					}
					var got bytes.Buffer
					got.Grow(want.Len())
					if _, err := io.Copy(&got, r); err != nil {
						t.Fatalf("copy: %v", err)
					}
					if err := r.Close(); err != nil {
						t.Fatalf("close: %v", err)
					}
					return bytes.Equal(got.Bytes(), want.Bytes())
				}
			}
			if err := quick.Check(check(tc.Kind), &quick.Config{Values: tc.Values}); err != nil {
				asCheckError(t, err)
			}
		})
	}
}

func TestShortRead(t *testing.T) {
	want := []byte("\xFF\xFF")
	z, d, err := Detect(bytes.NewReader(want))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = z.Close()
	}()
	if got, want := d, KindNone; got != want {
		t.Errorf("wrong compression? got: %v, want %v", got, want)
	}
	got, err := io.ReadAll(z)
	if err != nil {
		t.Errorf("read error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("bad roundtrip:\n%v\n%v", got, want)
	}
}

func asCheckError(t *testing.T, err error) {
	ce := err.(*quick.CheckError)
	hdr := ce.In[1].(*bytes.Buffer).Bytes()[:maxSz]
	t.Errorf("#%d: failed on input %v", ce.Count, hex.EncodeToString(hdr))
}

func makeBlobs[W io.WriteCloser](mk func(io.Writer) W) func([]reflect.Value, *rand.Rand) {
	const blobSize = 4096
	return func(vs []reflect.Value, rng *rand.Rand) {
		var orig, z bytes.Buffer

		wc := mk(&z)
		rd := io.LimitReader(rng, blobSize)
		if _, err := orig.ReadFrom(io.TeeReader(rd, wc)); err != nil {
			panic(err)
		}
		if err := wc.Close(); err != nil {
			panic(err)
		}

		vs[0] = reflect.ValueOf(&orig)
		vs[1] = reflect.ValueOf(&z)
	}
}

var blobGzip = makeBlobs(gzip.NewWriter)
var blobZstd = makeBlobs(func(w io.Writer) *zstd.Encoder {
	z, err := zstd.NewWriter(w)
	if err != nil {
		panic(err)
	}
	return z
})

var blobUncompressed = makeBlobs(func(w io.Writer) io.WriteCloser {
	r := struct {
		io.Writer
		io.Closer
	}{
		Writer: w,
		Closer: io.NopCloser(nil),
	}
	return &r
})
