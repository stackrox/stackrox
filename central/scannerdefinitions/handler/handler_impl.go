package handler

import (
	"io"
	"net/http"
	"os"
	"path"

	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc/codes"
)

const (
	scannerDefinitionsSubdir = `scannerdefinitions`
)

var (
	scannerDefinitionsPath = path.Join(migrations.DBMountPath, scannerDefinitionsSubdir)
	scannerDefinitionsFile = path.Join(scannerDefinitionsPath, "clair_definitions_central.sql.gz")
)

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		get(w, r)
		return
	}
	if r.Method == http.MethodPost {
		post(w, r)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func get(w http.ResponseWriter, r *http.Request) {
	exists, err := fileutils.Exists(scannerDefinitionsFile)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "couldn't check for scanner definitions file: %v", err)
		return
	}
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("No scanner definitions found"))
		return
	}
	file, err := os.Open(scannerDefinitionsFile)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "couldn't open file for reading: %v", err)
		return
	}
	defer utils.IgnoreError(file.Close)

	_, err = io.Copy(w, file)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "couldn't write file contents to response: %v", err)
		return
	}
	w.Header().Add("Content-Disposition", `attachment; filename="clair_definitions_central.sql.gz"`)
}

func post(w http.ResponseWriter, r *http.Request) {
	err := os.MkdirAll(scannerDefinitionsPath, 0755)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "failed to create directory: %v", err)
		return
	}

	file, err := os.Create(scannerDefinitionsFile)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "failed to create file: %v", err)
		return
	}
	defer utils.IgnoreError(file.Close)

	_, err = io.Copy(file, r.Body)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "failed to write to file: %v", err)
		return
	}
	_, _ = w.Write([]byte("Successfully stored the scanner definitions"))
}
