package index

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/quay/claircore"
)

type (
	// indexDiagnostics records observations without making capture errors fail indexing.
	indexDiagnostics struct {
		dir string
	}

	mappingCaptureTransport struct {
		base http.RoundTripper
		diag *indexDiagnostics
	}

	mappingCaptureBody struct {
		io.ReadCloser
		file *os.File
	}
)

func newIndexDiagnostics(cfg NodeIndexerConfig) *indexDiagnostics {
	if cfg.DebugDir == "" {
		return nil
	}
	if err := os.MkdirAll(cfg.DebugDir, 0700); err != nil {
		log.Warnf("Node index diagnostics: creating directory: %v", err)
		return nil
	}
	dir, err := os.MkdirTemp(cfg.DebugDir, "scan-"+time.Now().UTC().Format("20060102T150405Z")+"-")
	if err != nil {
		log.Warnf("Node index diagnostics: creating scan directory: %v", err)
		return nil
	}
	d := &indexDiagnostics{dir: dir}
	build, _ := debug.ReadBuildInfo()
	d.writeJSON("scan.json", map[string]any{
		"started_at": time.Now().UTC(), "node": os.Getenv("ROX_NODE_NAME"),
		"host_path": cfg.HostPath, "os_release_path": cfg.OSReleasePath,
		"package_db_filter": cfg.PackageDBFilter, "mapping_url": cfg.Repo2CPEMappingURL,
		"mapping_file": cfg.Repo2CPEMappingFile, "build": build,
	})
	log.Infof("Node index diagnostics: capturing scan at %s", dir)
	return d
}

func (d *indexDiagnostics) writeJSON(name string, value any) {
	if d == nil {
		return
	}
	b, err := json.MarshalIndent(value, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join(d.dir, name), b, 0600)
	}
	if err != nil {
		log.Warnf("Node index diagnostics: writing %s: %v", name, err)
	}
}

func (d *indexDiagnostics) captureFile(destination string, sys fs.FS, source string) string {
	f, err := sys.Open(source)
	if err != nil {
		return err.Error()
	}
	defer func() { _ = f.Close() }()
	path := filepath.Join(d.dir, destination)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err.Error()
	}
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err.Error()
	}
	_, copyErr := io.Copy(out, f)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr.Error()
	}
	if closeErr != nil {
		return closeErr.Error()
	}
	return "captured"
}

func (d *indexDiagnostics) captureInputs(name string, l *claircore.Layer) {
	if d == nil {
		return
	}
	sys, err := l.FS()
	if err != nil {
		d.writeJSON(name+"-error.json", err.Error())
		return
	}
	paths := []string{
		"etc/os-release", "usr/lib/os-release", "etc/system-release-cpe",
		"root/buildinfo/labels.json", "usr/share/buildinfo/labels.json",
		"var/lib/dnf/history.sqlite", "usr/lib/sysimage/libdnf5/transaction_history.sqlite",
	}
	for _, db := range paths[5:] {
		paths = append(paths, db+"-wal", db+"-shm")
	}
	// Preserve glob order because ClairCore selects the first match of the first nonempty glob.
	matches := make(map[string][]string)
	for _, glob := range []string{
		"root/buildinfo/content_manifests/*.json",
		"usr/share/buildinfo/content_manifests/*.json",
		"usr/share/buildinfo/*.json",
	} {
		found, err := fs.Glob(sys, glob)
		if err != nil {
			log.Warnf("Node index diagnostics: glob %s: %v", glob, err)
		}
		matches[glob] = found
		paths = append(paths, found...)
	}
	status := make(map[string]string)
	for _, path := range paths {
		status[path] = d.captureFile(filepath.Join(name, filepath.FromSlash(path)), sys, path)
	}
	d.writeJSON(name+"-files.json", status)
	d.writeJSON(name+"-manifest-globs.json", matches)
}

func (d *indexDiagnostics) captureAssignments(report *claircore.IndexReport) {
	if d == nil {
		return
	}
	assignments := make(map[string]any, len(report.Packages))
	for id, pkg := range report.Packages {
		hint, err := url.ParseQuery(pkg.RepositoryHint)
		rule := "all layer repositories: no repoid hint"
		if hint.Get("repoid") != "" {
			rule = "only repositories whose URI repoid equals the first package repoid hint"
		}
		entry := map[string]any{
			"name": pkg.Name, "version": pkg.Version, "arch": pkg.Arch,
			"package_db": pkg.PackageDB, "repository_hint": pkg.RepositoryHint,
			"repoid": hint.Get("repoid"), "rule": rule,
			"environments": report.Environments[id],
		}
		if err != nil {
			entry["hint_parse_error"] = err.Error()
		}
		var repos []*claircore.Repository
		for _, environment := range report.Environments[id] {
			for _, repoID := range environment.RepositoryIDs {
				repos = append(repos, report.Repositories[repoID])
			}
		}
		entry["assigned_repositories"] = repos
		assignments[id] = entry
	}
	d.writeJSON("rpm-assignments.json", assignments)
}

func (t *mappingCaptureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		t.diag.writeJSON("mapping-http-error.json", err.Error())
		return resp, err
	}
	t.diag.writeJSON("mapping-http.json", map[string]any{
		"url": req.URL.String(), "status": resp.StatusCode,
		"last_modified": resp.Header.Get("Last-Modified"), "etag": resp.Header.Get("ETag"),
	})
	f, err := os.OpenFile(filepath.Join(t.diag.dir, "mapping-response.json"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		log.Warnf("Node index diagnostics: capturing mapping response: %v", err)
		return resp, nil
	}
	resp.Body = &mappingCaptureBody{ReadCloser: resp.Body, file: f}
	return resp, nil
}

func (b *mappingCaptureBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if b.file != nil {
		if _, writeErr := b.file.Write(p[:n]); writeErr != nil {
			log.Warnf("Node index diagnostics: writing mapping response: %v", writeErr)
			_ = b.file.Close()
			b.file = nil
		}
	}
	return n, err
}

func (b *mappingCaptureBody) Close() error {
	if b.file != nil {
		if err := b.file.Close(); err != nil {
			log.Warnf("Node index diagnostics: closing mapping capture: %v", err)
		}
	}
	return b.ReadCloser.Close()
}
