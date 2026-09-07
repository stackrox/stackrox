package cli

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

var downloadPath = "/assets/downloads/cli"

// Handler for serving roxctl binaries from Central UI.
// Binaries are stored as .tar.gz and extracted on the fly on each request.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)
		if err := serveFromTarball(w, filename); err != nil {
			http.Error(w, "not found", http.StatusNotFound)
		}
	}
}

func serveFromTarball(w http.ResponseWriter, filename string) error {
	return serveFromDir(w, downloadPath, filename)
}

func serveFromDir(w http.ResponseWriter, dir, filename string) error {
	tarPath := filepath.Join(dir, filename+".tar.gz")
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err != nil {
			return fmt.Errorf("entry %q not found in tarball", filename)
		}
		if hdr.Name == filename {
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", hdr.Size))
			_, err = io.Copy(w, tr)
			return err
		}
	}
}
