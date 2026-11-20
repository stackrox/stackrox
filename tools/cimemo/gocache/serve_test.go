package gocache

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGo drives Serve the way cmd/go does: JSON requests on stdin,
// JSON responses (+ base64 put bodies) on stdout.
type fakeGo struct {
	in        bytes.Buffer
	out       strings.Builder
	cacheDir  string
	statsPath string
}

func (f *fakeGo) send(v interface{}) {
	json.NewEncoder(&f.in).Encode(v)
}

func (f *fakeGo) sendPutBody(body []byte) {
	json.NewEncoder(&f.in).Encode(base64.StdEncoding.EncodeToString(body))
}

func (f *fakeGo) run(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	f.cacheDir = dir
	f.statsPath = filepath.Join(dir, "stats.json")
	if err := serveLoop(bytes.NewReader(f.in.Bytes()), &f.out, dir, f.statsPath, nil); err != nil {
		t.Fatalf("serve: %v", err)
	}
}

func (f *fakeGo) responses(t *testing.T) []Response {
	t.Helper()
	var out []Response
	dec := json.NewDecoder(strings.NewReader(f.out.String()))
	for {
		var r Response
		if err := dec.Decode(&r); err != nil {
			break
		}
		out = append(out, r)
	}
	return out
}

func TestProtocolPutGetHit(t *testing.T) {
	f := &fakeGo{}
	actionID := bytes32("action-1")
	outputBody := []byte("compiled package output bytes")
	outputID := sha256Bytes(outputBody) // faithful: go's OutputID is the content hash

	f.send(Request{ID: 1, Command: "put", ActionID: actionID, OutputID: outputID, BodySize: int64(len(outputBody))})
	f.sendPutBody(outputBody)
	f.send(Request{ID: 2, Command: "get", ActionID: actionID})
	f.send(Request{ID: 3, Command: "close"})
	f.run(t)

	resps := f.responses(t)
	if len(resps) != 4 { // handshake + 3
		t.Fatalf("expected 4 responses, got %d: %+v", len(resps), resps)
	}
	if len(resps[0].KnownCommands) == 0 {
		t.Fatal("expected handshake with KnownCommands")
	}
	if resps[1].ID != 1 {
		t.Fatalf("put response ID: %+v", resps[1])
	}
	if resps[1].DiskPath == "" || resps[1].Size != int64(len(outputBody)) {
		t.Fatalf("put response must carry DiskPath+Size (go 1.27 requires it): %+v", resps[1])
	}
	get := resps[2]
	if get.Miss {
		t.Fatal("get after put should hit")
	}
	if get.ID != 2 || get.Size != int64(len(outputBody)) {
		t.Fatalf("get response: %+v", get)
	}
	if !bytes.Equal(get.OutputID, outputID) {
		t.Fatalf("outputID roundtrip: %x", get.OutputID)
	}
	// DiskPath must point at a file with the exact content.
	if get.DiskPath == "" {
		t.Fatal("expected DiskPath in get response")
	}
	data, err := os.ReadFile(get.DiskPath)
	if err != nil || !bytes.Equal(data, outputBody) {
		t.Fatalf("DiskPath content mismatch: %v %q", err, data)
	}
}

func TestProtocolGetMiss(t *testing.T) {
	f := &fakeGo{}
	f.send(Request{ID: 7, Command: "get", ActionID: bytes32("unknown")})
	f.send(Request{ID: 8, Command: "close"})
	f.run(t)

	resps := f.responses(t)
	if !resps[1].Miss {
		t.Fatalf("unknown action should miss: %+v", resps[1])
	}
	if resps[1].ID != 7 {
		t.Fatalf("miss response ID: %+v", resps[1])
	}
}

func TestStatsWritten(t *testing.T) {
	f := &fakeGo{}
	f.send(Request{ID: 1, Command: "get", ActionID: bytes32("nope")})
	f.send(Request{ID: 2, Command: "put", ActionID: bytes32("a"), OutputID: bytes32("o"), BodySize: 3})
	f.sendPutBody([]byte("xyz"))
	f.send(Request{ID: 3, Command: "close"})
	f.run(t)

	s, err := ReadStats(f.statsPath)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if s.Gets != 1 || s.Misses != 1 || s.Hits != 0 {
		t.Fatalf("get stats: %+v", s)
	}
	if s.Puts != 1 || s.PutBytes != 3 {
		t.Fatalf("put stats: %+v", s)
	}
	if len(s.MissedActions) != 1 || s.MissedActions[0] != hexOf("nope") {
		t.Fatalf("missed actions: %+v", s.MissedActions)
	}
}

func TestStandardLayoutCohabitation(t *testing.T) {
	// What the proxy writes must be readable by go's own cache logic:
	// the -a record format and -d file naming must match exactly what
	// go writes natively.
	f := &fakeGo{}
	body := []byte("native-format check")
	f.send(Request{ID: 1, Command: "put", ActionID: bytes32("cohab"), OutputID: sha256Bytes(body), BodySize: int64(len(body))})
	f.sendPutBody(body)
	f.send(Request{ID: 2, Command: "close"})
	f.run(t)

	cacheDir := f.cacheDir
	// -d file named by outputID (= sha256 of body).
	d := dataPath(cacheDir, hex.EncodeToString(sha256Bytes(body)))
	if data, err := os.ReadFile(d); err != nil || !bytes.Equal(data, body) {
		t.Fatalf("data file not in standard layout: %v", err)
	}
	// -a record parseable by readActionRecord (our mirror of go's reader).
	outID, size, ok := readActionRecord(cacheDir, hexOf("cohab"))
	if !ok || size != int64(len(body)) || outID != hex.EncodeToString(sha256Bytes(body)) {
		t.Fatalf("action record: %q %d %v", outID, size, ok)
	}
}

// --- helpers ---

func bytes32(s string) []byte {
	// deterministic 32-byte pseudo-hash for tests
	b := make([]byte, 32)
	for i := 0; i < len(s); i++ {
		b[i%32] ^= s[i]
	}
	return b
}

func hexOf(s string) string {
	return hex.EncodeToString(bytes32(s))
}

func sha256Bytes(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
}
