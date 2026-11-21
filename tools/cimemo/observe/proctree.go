// Linux-native observation: poll /proc for ALL descendants of the
// build process and read their open fds directly. lsof on the top PID
// only sees the orchestrator (cargo/make) — workers (rustc, clang)
// do the actual source reads and are children. /proc fd polling is
// also much faster than spawning lsof, catching shorter opens.
package observe

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TrackProcTree monitors every descendant of pid via /proc. Falls back
// silently (no-op) on non-Linux or missing /proc.
func (r *Reads) TrackProcTree(pid int, poll time.Duration) {
	if _, err := os.Stat("/proc"); err != nil {
		return
	}
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for range ticker.C {
		pids := descendants(pid)
		if pids == nil && !processAlive(pid) {
			return // root exited and no descendants seen
		}
		alive := false
		for _, p := range append([]int{pid}, pids...) {
			if processAlive(p) {
				alive = true
			}
			r.pollProcFDs(p)
		}
		if !alive {
			return
		}
	}
}

func processAlive(pid int) bool {
	_, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
	return err == nil
}

// descendants walks /proc/*/stat PPid chains to find all live
// descendants of root. Returns nil when /proc is unavailable.
func descendants(root int) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	parent := map[int]int{}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", e.Name(), "stat"))
		if err != nil {
			continue
		}
		// stat: "pid (comm) state ppid ..." — comm may contain spaces
		// and parens; find the LAST ')' and parse after it.
		s := string(data)
		i := strings.LastIndexByte(s, ')')
		if i < 0 || i+2 > len(s) {
			continue
		}
		fields := strings.Fields(s[i+2:])
		if len(fields) < 2 {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		parent[pid] = ppid
	}
	var out []int
	for pid := range parent {
		p, seen := pid, 0
		for p != root && p > 1 && seen < 64 {
			pp, ok := parent[p]
			if !ok {
				break
			}
			p = pp
			seen++
		}
		if p == root && pid != root {
			out = append(out, pid)
		}
	}
	return out
}

// pollProcFDs records open files of one process via /proc/<pid>/fd.
func (r *Reads) pollProcFDs(pid int) {
	dir := filepath.Join("/proc", strconv.Itoa(pid), "fd")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range entries {
		link, err := os.Readlink(filepath.Join(dir, e.Name()))
		if err != nil || !strings.HasPrefix(link, "/") {
			continue
		}
		r.files[link] = true
	}
}
