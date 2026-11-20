package observe

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestTrackOpenFiles(t *testing.T) {
	// Create a test script that opens a file and holds it
	dir := t.TempDir()
	testFile := filepath.Join(dir, "testfile.txt")
	os.WriteFile(testFile, []byte("test content"), 0o644)

	// Create a script that opens the file, sleeps, then exits
	script := filepath.Join(dir, "hold_open.sh")
	os.WriteFile(script, []byte("#!/bin/bash\nexec 3<"+testFile+"\nsleep 1\n"), 0o755)

	reads := NewReads()
	cmd := exec.Command(script)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Track file descriptors while the process runs
	go reads.Track(cmd.Process.Pid, 50*time.Millisecond)

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond) // let the last poll complete

	files := reads.Files(dir)
	found := false
	for _, f := range files {
		if f == "testfile.txt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected testfile.txt in observed files, got %v", files)
	}
}

func TestTrackShortLivedProcess(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "quick.txt")
	os.WriteFile(testFile, []byte("quick"), 0o644)

	reads := NewReads()
	cmd := exec.Command("cat", testFile)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	go reads.Track(cmd.Process.Pid, 50*time.Millisecond)
	cmd.Wait()
	time.Sleep(100 * time.Millisecond)

	// May or may not catch it (depends on timing) — just verify no crash
	files := reads.Files(dir)
	t.Logf("observed %d files (short-lived process may be missed)", len(files))
}

func TestFilesRelativeToWorkDir(t *testing.T) {
	r := NewReads()
	r.mu.Lock()
	r.files["/some/dir/sub/file.go"] = true
	r.files["/other/place/file.go"] = true
	r.mu.Unlock()

	rel := r.Files("/some/dir")
	if len(rel) != 1 || rel[0] != "sub/file.go" {
		t.Fatalf("expected [sub/file.go], got %v", rel)
	}
}
