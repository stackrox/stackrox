// Package observe discovers which files a command actually reads,
// solving the key-closure problem without manual configuration.
//
// On macOS: polls the child process's file descriptors via lsof.
// On Linux: same approach (lsof or /proc/PID/fd).
//
// This catches files held open during compilation, parsing, etc.
// Short-lived opens (< poll interval) may be missed — for Go builds,
// the compile cache (GOCACHEPROG) provides exact per-package tracking.
package observe

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Reads tracks files opened by a command during its execution.
type Reads struct {
	mu    sync.Mutex
	files map[string]bool // set of absolute paths read
}

// NewReads creates a file-read tracker.
func NewReads() *Reads {
	return &Reads{files: make(map[string]bool)}
}

// Track monitors a process's open file descriptors while it runs.
// Call this in a goroutine before calling cmd.Wait().
// The poll interval trades completeness for overhead.
func (r *Reads) Track(pid int, poll time.Duration) {
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for range ticker.C {
		if !r.pollFDs(pid) {
			return // process exited
		}
	}
}

// pollFDs runs lsof on the process and records file paths.
// Returns false if the process is no longer running.
func (r *Reads) pollFDs(pid int) bool {
	// lsof -p PID -Fn : list open files, name-only mode (-Fn prefix)
	// -w : suppress warnings
	out, err := exec.Command("lsof", "-p", strconv.Itoa(pid), "-Fn", "-w").Output()
	if err != nil {
		// Process exited (or lsof failed)
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") && len(line) > 1 {
			path := line[1:]
			// Only track regular files in the working directory
			// (skip sockets, pipes, devices, system files)
			if isProjectFile(path) {
				r.files[path] = true
			}
		}
	}
	return true
}

// Files returns all file paths that were observed as open.
// Paths are relative to workDir.
func (r *Reads) Files(workDir string) []string {
	// Resolve symlinks in workDir to match lsof's resolved paths
	// (macOS: /var → /private/var, /tmp → /private/tmp)
	real, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		real = workDir
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for path := range r.files {
		// Skip directories (cwd shows up as a held "read") and
		// dot-files (lock files, transient state — cargo's
		// .cargo-*-lock poisoned read-sets in the fact case study).
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			continue
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			continue
		}
		if rel, err := filepath.Rel(real, path); err == nil && !strings.HasPrefix(rel, "..") {
			out = append(out, rel)
		}
	}
	return out
}

// isProjectFile filters out non-project files.
func isProjectFile(path string) bool {
	// Skip devices, sockets, pipes (lsof reports these with special paths)
	if strings.HasPrefix(path, "/dev/") || strings.HasPrefix(path, "/proc/") {
		return false
	}
	// Skip system binaries and libraries (not project source)
	for _, skip := range []string{"/usr/", "/System/", "/Library/", "/opt/", "/etc/"} {
		if strings.HasPrefix(path, skip) {
			return false
		}
	}
	// NOTE: /private/ prefix is NOT filtered — on macOS, /tmp and /var
	// resolve to /private/tmp and /private/var. The project-dir check
	// in Files() handles scoping. We only skip known system dirs above.
	return true
}
