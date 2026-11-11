package main

// Protocol-level tests: spawn gocache-serve as a subprocess and speak the
// GOCACHEPROG wire protocol exactly as cmd/go does — JSON request lines
// on stdin, with put bodies as quoted-base64 lines after the JSON, and
// JSON response lines on stdout. Uses the dir backend (no Azure, no
// credentials, runs anywhere).
//
// This tests the layer where every real bug has lived: meta encoding,
// the SIGBUS atomic-write fix, phantom entries (empty-body puts), the
// in-flight map, and the OutputID round-trip.

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Request mirrors the GOCACHEPROG wire format for the test.
type protoReq struct {
	ID       int64  `json:"id"`
	Command  string `json:"command"`
	ActionID []byte `json:"actionID,omitempty"`
	OutputID []byte `json:"outputID,omitempty"`
	BodySize int64  `json:"bodySize,omitempty"`
	DiskPath string `json:"diskPath,omitempty"`
}

type protoResp struct {
	ID            int64    `json:"id"`
	Err           string   `json:"err,omitempty"`
	Miss          bool     `json:"miss,omitempty"`
	DiskPath      string   `json:"diskPath,omitempty"`
	OutputID      []byte   `json:"outputID,omitempty"`
	Size          int64    `json:"size,omitempty"`
	KnownCommands []string `json:"KnownCommands,omitempty"`
}

// protoSession manages a gocache-serve subprocess speaking GOCACHEPROG.
type protoSession struct {
	t      *testing.T
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr *strings.Builder
	nextID int64
}

func startProtoSession(t *testing.T, storeDir string) *protoSession {
	t.Helper()

	// Build gocache-serve (or reuse a pre-built one via env)
	binPath := os.Getenv("GOCACHE_TEST_BIN")
	if binPath == "" {
		tmp := t.TempDir()
		binPath = filepath.Join(tmp, "gocache-serve")
		out, err := exec.Command("go", "build", "-o", binPath, ".").CombinedOutput()
		if err != nil {
			t.Fatalf("build gocache-serve: %v\n%s", err, out)
		}
	}

	localDir := t.TempDir()
	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"GOCACHE_SERVE_BACKEND=dir",
		"GOCACHE_SERVE_DIR_STORE="+storeDir,
		"GOCACHE_SERVE_DIR="+localDir,
		"GOCACHE_SERVE_MANIFEST=prototest", // enable manifest for lifecycle tests
		"GOCACHE_SERVE_RUN_ID=test-run",
		"GOCACHE_SERVE_PREFETCH=0",
		"GOCACHE_SERVE_GATE=0",
	)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := &strings.Builder{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start gocache-serve: %v (stderr: %s)", err, stderr.String())
	}

	ps := &protoSession{
		t:      t,
		cmd:    cmd,
		stdin:  stdinPipe,
		stdout: bufio.NewScanner(stdoutPipe),
		stderr: stderr,
	}

	// Read the capability handshake (ID 0, KnownCommands populated)
	var resp protoResp
	if !ps.stdout.Scan() {
		t.Fatalf("no handshake; stderr: %s", stderr.String())
	}
	if err := json.Unmarshal(ps.stdout.Bytes(), &resp); err != nil {
		t.Fatalf("handshake parse: %v\nraw: %s", err, ps.stdout.Bytes())
	}
	if resp.ID != 0 || len(resp.KnownCommands) == 0 {
		t.Fatalf("bad handshake: %+v", resp)
	}
	return ps
}

// send writes a request and reads the response. For puts with bodies,
// writes the quoted-base64 body line after the JSON, matching cmd/go's
// writeToChild exactly.
func (ps *protoSession) send(req *protoReq, body []byte) protoResp {
	ps.t.Helper()
	ps.nextID++
	req.ID = ps.nextID

	// Encode the JSON request line
	jsonBytes, err := json.Marshal(req)
	if err != nil {
		ps.t.Fatalf("marshal request: %v", err)
	}
	if _, err := ps.stdin.Write(jsonBytes); err != nil {
		ps.t.Fatalf("write request: %v", err)
	}
	if _, err := ps.stdin.Write([]byte("\n")); err != nil {
		ps.t.Fatalf("write newline: %v", err)
	}

	// Write the body line for puts (matches writeToChild in prog.go):
	// `"base64content"\n` as raw bytes
	if req.Command == "put" && req.BodySize > 0 && len(body) > 0 {
		encoded := base64.StdEncoding.EncodeToString(body)
		line := `"` + encoded + `"` + "\n"
		if _, err := ps.stdin.Write([]byte(line)); err != nil {
			ps.t.Fatalf("write body: %v", err)
		}
	}

	// Read the response
	if !ps.stdout.Scan() {
		ps.t.Fatalf("no response; stderr: %s", ps.stderr.String())
	}
	var resp protoResp
	if err := json.Unmarshal(ps.stdout.Bytes(), &resp); err != nil {
		ps.t.Fatalf("response parse: %v\nraw: %s", err, ps.stdout.Bytes())
	}
	if resp.ID != req.ID {
		ps.t.Fatalf("response ID %d, expected %d", resp.ID, req.ID)
	}
	return resp
}

func (ps *protoSession) close() {
	ps.t.Helper()
	ps.send(&protoReq{Command: "close"}, nil)
	ps.stdin.Close()
	ps.cmd.Wait()
}

// --- Tests ---

// TestProtocolPutGetRoundTrip: put an entry, close, restart the process,
// get it back. The fundamental cache contract.
func TestProtocolPutGetRoundTrip(t *testing.T) {
	storeDir := t.TempDir()
	actionID := []byte("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	outputID := make([]byte, 32)
	for i := range outputID {
		outputID[i] = byte(i)
	}
	body := []byte("test entry body content — with unicode: hello wörld")

	// Session 1: put
	ps := startProtoSession(t, storeDir)
	resp := ps.send(&protoReq{
		Command:  "put",
		ActionID: actionID,
		OutputID: outputID,
		BodySize: int64(len(body)),
	}, body)
	if resp.Err != "" {
		t.Fatalf("put error: %s", resp.Err)
	}
	if resp.DiskPath == "" {
		t.Fatal("put returned empty DiskPath")
	}
	ps.close()

	// Session 2: get (fresh process, same store)
	ps2 := startProtoSession(t, storeDir)
	resp2 := ps2.send(&protoReq{
		Command:  "get",
		ActionID: actionID,
	}, nil)
	if resp2.Err != "" {
		t.Fatalf("get error: %s", resp2.Err)
	}
	if resp2.Miss {
		t.Fatal("get returned Miss after put — entry lost across restart")
	}
	if resp2.DiskPath == "" {
		t.Fatal("get returned empty DiskPath")
	}
	// Verify the file content
	fileBody, err := os.ReadFile(resp2.DiskPath)
	if err != nil {
		t.Fatalf("read served file: %v", err)
	}
	if string(fileBody) != string(body) {
		t.Fatalf("body mismatch:\n put  %q\n got  %q", body, fileBody)
	}
	// Verify OutputID round-trip
	if len(resp2.OutputID) != 32 {
		t.Fatalf("OutputID length %d, expected 32", len(resp2.OutputID))
	}
	for i, b := range resp2.OutputID {
		if b != outputID[i] {
			t.Fatalf("OutputID byte %d: got %x expected %x", i, b, outputID[i])
		}
	}
	ps2.close()
}

// TestProtocolGetMiss: nonexistent entry returns Miss.
func TestProtocolGetMiss(t *testing.T) {
	storeDir := t.TempDir()
	ps := startProtoSession(t, storeDir)
	defer ps.close()

	resp := ps.send(&protoReq{
		Command:  "get",
		ActionID: []byte(strings.Repeat("0", 64)),
	}, nil)
	if !resp.Miss {
		t.Fatal("expected Miss for nonexistent entry")
	}
}

// TestProtocolEmptyPut: the phantom fix — zero-length bodies must be
// stored and served as hits with Size 0.
func TestProtocolEmptyPut(t *testing.T) {
	storeDir := t.TempDir()
	actionID := []byte(strings.Repeat("1", 64))
	outputID := make([]byte, 32)
	outputID[0], outputID[1], outputID[2] = 1, 2, 3

	// Session 1: put with empty body
	ps := startProtoSession(t, storeDir)
	resp := ps.send(&protoReq{
		Command:  "put",
		ActionID: actionID,
		OutputID: outputID,
		BodySize: 0, // empty — no body line follows
	}, nil)
	if resp.Err != "" {
		t.Fatalf("put error: %s", resp.Err)
	}
	ps.close()

	// Session 2: get — must be a hit (the phantom fix)
	ps2 := startProtoSession(t, storeDir)
	resp2 := ps2.send(&protoReq{
		Command:  "get",
		ActionID: actionID,
	}, nil)
	if resp2.Miss {
		t.Fatal("empty-body entry should be a hit after put — the phantom fix")
	}
	if resp2.Size != 0 {
		t.Fatalf("empty entry Size should be 0, got %d", resp2.Size)
	}
	ps2.close()
}

// TestProtocolLargeBody: entries at the boundary sizes (1KB, 100KB, 1MB)
// survive the round-trip intact.
func TestProtocolLargeBody(t *testing.T) {
	storeDir := t.TempDir()

	sizes := []int{1024, 100 * 1024, 1024 * 1024}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dKB", size/1024), func(t *testing.T) {
			actionID := []byte(fmt.Sprintf("%064d", size))
			body := make([]byte, size)
			for i := range body {
				body[i] = byte(i % 251)
			}

			ps := startProtoSession(t, storeDir)
			ps.send(&protoReq{
				Command:  "put",
				ActionID: actionID,
				OutputID: make([]byte, 32),
				BodySize: int64(size),
			}, body)
			ps.close()

			ps2 := startProtoSession(t, storeDir)
			resp := ps2.send(&protoReq{
				Command:  "get",
				ActionID: actionID,
			}, nil)
			if resp.Miss {
				t.Fatalf("%dKB entry lost", size/1024)
			}
			fileBody, err := os.ReadFile(resp.DiskPath)
			if err != nil {
				t.Fatal(err)
			}
			if len(fileBody) != size {
				t.Fatalf("size mismatch: put %d got %d", size, len(fileBody))
			}
			for i := 0; i < len(body); i++ {
				if fileBody[i] != body[i] {
					t.Fatalf("byte %d differs: %x vs %x", i, fileBody[i], body[i])
				}
			}
			ps2.close()
		})
	}
}

// TestProtocolMultipleEntries: 20 entries across a restart.
func TestProtocolMultipleEntries(t *testing.T) {
	storeDir := t.TempDir()

	ps1 := startProtoSession(t, storeDir)
	for i := 0; i < 20; i++ {
		id := []byte(fmt.Sprintf("%064d", i))
		body := []byte(fmt.Sprintf("body-for-entry-%d", i))
		ps1.send(&protoReq{
			Command:  "put",
			ActionID: id,
			OutputID: make([]byte, 32),
			BodySize: int64(len(body)),
		}, body)
	}
	ps1.close()

	ps2 := startProtoSession(t, storeDir)
	for i := 0; i < 20; i++ {
		id := []byte(fmt.Sprintf("%064d", i))
		resp := ps2.send(&protoReq{
			Command:  "get",
			ActionID: id,
		}, nil)
		if resp.Miss {
			t.Fatalf("entry %d lost after restart", i)
		}
	}
	ps2.close()
}

// TestProtocolSameSessionGetAfterPut: a get immediately after a put in
// the same session (no restart) must hit from local cache.
func TestProtocolSameSessionGetAfterPut(t *testing.T) {
	storeDir := t.TempDir()
	actionID := []byte(strings.Repeat("2", 64))
	body := []byte("immediate")

	ps := startProtoSession(t, storeDir)
	defer ps.close()

	ps.send(&protoReq{
		Command:  "put",
		ActionID: actionID,
		OutputID: make([]byte, 32),
		BodySize: int64(len(body)),
	}, body)

	resp := ps.send(&protoReq{
		Command:  "get",
		ActionID: actionID,
	}, nil)
	if resp.Miss {
		t.Fatal("get immediately after put in same session should hit")
	}
}

// TestProtocolCloseIsIdempotent: close responds cleanly.
func TestProtocolCloseResponse(t *testing.T) {
	storeDir := t.TempDir()
	ps := startProtoSession(t, storeDir)

	resp := ps.send(&protoReq{Command: "close"}, nil)
	if resp.Err != "" {
		t.Fatalf("close error: %s", resp.Err)
	}
	ps.stdin.Close()
	ps.cmd.Wait()
}

// TestProtocolCLIRoundTrip: the -store/-fetch CLI subcommands.
func TestProtocolCLIRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("builds gocache-serve; skip in -short")
	}
	binPath := os.Getenv("GOCACHE_TEST_BIN")
	if binPath == "" {
		tmp := t.TempDir()
		binPath = filepath.Join(tmp, "gocache-serve")
		out, err := exec.Command("go", "build", "-o", binPath, ".").CombinedOutput()
		if err != nil {
			t.Fatalf("build: %v\n%s", err, out)
		}
	}

	storeDir := t.TempDir()
	dirEnv := func() []string {
		return append(os.Environ(),
			"GOCACHE_SERVE_BACKEND=dir",
			"GOCACHE_SERVE_DIR_STORE="+storeDir)
	}

	// -store
	srcFile := filepath.Join(t.TempDir(), "data")
	os.WriteFile(srcFile, []byte("hello protocol world"), 0o644)
	cmd := exec.Command(binPath, "-store", "cli-test-key", srcFile)
	cmd.Env = dirEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("-store: %v\n%s", err, out)
	}

	// -fetch
	dstFile := filepath.Join(t.TempDir(), "fetched")
	cmd = exec.Command(binPath, "-fetch", "cli-test-key", dstFile)
	cmd.Env = dirEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("-fetch: %v\n%s", err, out)
	}
	got, _ := os.ReadFile(dstFile)
	if string(got) != "hello protocol world" {
		t.Fatalf("CLI round-trip mismatch: %q", string(got))
	}

	// -fetch for missing key fails
	cmd = exec.Command(binPath, "-fetch", "nonexistent-key", dstFile)
	cmd.Env = dirEnv()
	if err := cmd.Run(); err == nil {
		t.Fatal("-fetch for missing key should fail")
	}
}
