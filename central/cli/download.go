package cli

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

const downloadPath = "/assets/downloads/cli"

var log = logging.LoggerForModule()

// Handler serves roxctl binaries from Central UI.
// Binaries are stored as .tar.gz and extracted on the fly on each request.
func Handler() http.HandlerFunc {
	return handlerWithDir(downloadPath)
}

func handlerWithDir(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
			http.Error(w, fmt.Sprintf("method %s not allowed", r.Method), http.StatusMethodNotAllowed)
			return
		}

		filename := filepath.Base(r.URL.Path)
		tarPath := filepath.Join(dir, strings.TrimSuffix(filename, ".exe")+".tar.gz")

		f, err := os.Open(tarPath)
		if err != nil {
			log.Warnf("error opening tarball for requested binary %q: %v", filename, err)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer utils.IgnoreError(f.Close)

		gz, err := gzip.NewReader(f)
		if err != nil {
			log.Warnf("error opening gzip reader for %s: %v", tarPath, err)
			http.Error(w, "invalid archive", http.StatusInternalServerError)
			return
		}
		defer utils.IgnoreError(gz.Close)

		tr := tar.NewReader(gz)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				log.Warnf("binary %q not found in %s; tarball entry name may not match the requested filename", filename, tarPath)
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if err != nil {
				log.Warnf("error reading archive %s: %v", tarPath, err)
				http.Error(w, "error reading archive", http.StatusInternalServerError)
				return
			}
			if hdr.Name != filename {
				continue
			}
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", strconv.FormatInt(hdr.Size, 10))
			w.Header().Set("Accept-Ranges", "none")
			w.Header().Set("Cache-Control", "no-cache")
			if r.Method == http.MethodGet {
				if _, err = io.Copy(w, tr); err != nil {
					log.Warnf("failed to stream %s: %v", filename, err)
					panic(http.ErrAbortHandler)
				}
			}
			return
		}
	}
}
