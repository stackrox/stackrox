package main

// Streaming large-blob paths for the CLI store/fetch subcommands.
//
// The original whole-buffer path (os.ReadFile + gzipBytes + Put) peaks at
// ~2.5x the blob size in RAM: for the 1.5GB roxctl-all tar that is ~4GB,
// which OOM-killed GitHub runners mid-upload ("runner has received a
// shutdown signal", exit 137). These paths keep peak memory bounded
// (a few MB of buffers) by streaming: file → gzip temp file → HTTP body,
// and HTTP body → gunzip → destination file, hashing along the way.
//
// Marker semantics are identical to the buffered path: the .sha256 marker
// is the hash of the LOGICAL (decompressed) content; the blob at rest is
// gzip (level 1) unless the source already was gzip.

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// largeBlobThreshold: blobs at or above this size use the streaming
// paths; smaller ones keep the in-memory fan-out (concurrent replica
// writes from one buffer).
const largeBlobThreshold = 64 << 20 // 64 MiB

// fileSHA256 hashes a file streaming. Returns hex digest and size.
func fileSHA256(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// isGzipFile peeks the first two bytes for the gzip magic.
func isGzipFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	var magic [2]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return false, nil // empty or 1-byte file: not gzip
		}
		return false, err
	}
	return magic[0] == 0x1f && magic[1] == 0x8b, nil
}

// gzipToFileStreaming compresses src into a new temp file next to dst,
// returning its path (caller removes). Bounded memory: gzip internal
// buffers only.
func gzipToFileStreaming(src string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(src), ".gz-stream-*")
	if err != nil {
		return "", err
	}
	zw, err := gzip.NewWriterLevel(tmp, gzip.BestSpeed)
	if err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	if _, err := io.Copy(zw, in); err != nil {
		zw.Close()
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	if err := zw.Close(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

// putFileStream uploads a file to the blob store, gzipping at rest.
// Logical (pre-gzip) content determines the at-rest bytes: files that
// are already gzip pass through untouched, like gzipBytes.
// Returns the hash of the logical content (the marker value).
func putFileStream(a *AzureBackend, key, path string) (string, error) {
	gz, err := isGzipFile(path)
	if err != nil {
		return "", err
	}
	uploadPath := path
	if !gz {
		tmp, err := gzipToFileStreaming(path)
		if err != nil {
			return "", err
		}
		defer os.Remove(tmp)
		uploadPath = tmp
	}
	// Hash the LOGICAL content (the source file, pre-gzip).
	sum, _, err := fileSHA256(path)
	if err != nil {
		return "", err
	}
	if err := a.putFileBlob(key, uploadPath); err != nil {
		return "", err
	}
	return sum, nil
}

// putFileBlob issues a single-shot Put Blob streaming from the file.
func (a *AzureBackend) putFileBlob(key, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	token, err := a.getToken()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", a.blobURL(key), f)
	if err != nil {
		return err
	}
	req.ContentLength = st.Size()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", "2023-11-03")
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("put stream %s: %w", key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("put stream %s: %s %s", key, resp.Status, body)
	}
	return nil
}

// getFileStream downloads a blob to dest, gunzipping if at-rest gzip.
// Returns the hash of the logical content written; found=false on 404
// (the backend's Get convention is (nil, nil)).
func getFileStream(a *AzureBackend, key, dest string) (string, bool, error) {
	token, err := a.getToken()
	if err != nil {
		return "", false, err
	}
	req, err := http.NewRequest("GET", a.blobURL(key), nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-version", "2023-11-03")
	resp, err := a.client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return "", false, nil
	}
	if resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("get stream %s: %s", key, resp.Status)
	}
	out, err := os.Create(dest)
	if err != nil {
		return "", false, err
	}
	defer out.Close()
	h := sha256.New()
	// Peek magic to decide gunzip; reprepend the peeked bytes.
	var src io.Reader = resp.Body
	magic := make([]byte, 2)
	n, _ := io.ReadFull(io.LimitReader(resp.Body, 2), magic)
	if n == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		zr, err := gzip.NewReader(io.MultiReader(bytes.NewReader(magic), resp.Body))
		if err != nil {
			return "", false, err
		}
		src = zr
	} else {
		src = io.MultiReader(bytes.NewReader(magic[:n]), resp.Body)
	}
	if _, err := io.Copy(io.MultiWriter(out, h), src); err != nil {
		return "", false, err
	}
	return hex.EncodeToString(h.Sum(nil)), true, nil
}
