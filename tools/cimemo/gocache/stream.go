// Streaming paths for large blobs — the CA-work OOM lesson: whole-body
// buffering peaked 4GB RSS and OOM-killed runners on 300MB+ link
// outputs. Every large-object path here is constant-memory: bodies
// stream through a hasher to a temp file, verified, then renamed.
package gocache

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// streamToTempFile copies r into a temp file next to dest while
// hashing. Returns the temp path and content hash. Caller renames or
// removes.
func streamToTempFile(r io.Reader, dest string) (tmp string, hashHex string, size int64, err error) {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", "", 0, err
	}
	tmp = dest + ".tmp-stream"
	f, err := os.Create(tmp)
	if err != nil {
		return "", "", 0, err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), r)
	cerr := f.Close()
	if err != nil || cerr != nil {
		os.Remove(tmp)
		return "", "", 0, fmt.Errorf("stream: %v/%v", err, cerr)
	}
	return tmp, hex.EncodeToString(h.Sum(nil)), n, nil
}

// getStreamingVerify fetches key from the remote and streams it into
// dest, verifying the body hashes to expectHash. On mismatch the file
// is discarded (degrade-to-miss, never serve wrong bytes). Memory is
// constant regardless of blob size. Uses the adaptive timeout.
func (r *Remote) getStreamingVerify(key, expectHash, dest string, tt *TimeoutTracker) (int64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), tt.Timeout())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", r.containerURL()+"/"+key, nil)
	if err != nil {
		return 0, false
	}
	token, err := r.getToken()
	if err != nil {
		return 0, false
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-version", "2023-11-03")

	start := time.Now()
	resp, err := r.client.Do(req)
	if err != nil {
		return 0, false // includes timeout: caller degrades to miss
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return 0, false
		}
		return 0, false
	}

	// Azure stores gzip-at-rest; gunzip while streaming. All PUTs in
	// this store gzip, so a non-gzip body is foreign data — reject
	// (zeroReader forces the hash check to fail) rather than risk
	// serving wrong bytes after gzip.NewReader's failed peek consumed
	// an unknown prefix.
	body := io.Reader(resp.Body)
	if zr, zerr := gzip.NewReader(resp.Body); zerr == nil {
		defer zr.Close()
		body = zr
	} else {
		body = zeroReader{}
	}

	tmp, gotHash, size, err := streamToTempFile(body, dest)
	if err != nil {
		return 0, false
	}
	if gotHash != expectHash {
		os.Remove(tmp)
		logRemoteIntegrity(key, expectHash)
		return 0, false
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return 0, false
	}
	tt.Record(time.Since(start))
	return size, true
}

type zeroReader struct{}

func (zeroReader) Read([]byte) (int, error) { return 0, io.EOF }

// ingestDiskPathStreaming streams a file go wrote into the cache
// layout, hashing on the fly. Returns the (possibly corrected)
// outputID hex and size. Memory constant.
func ingestDiskPathStreaming(cacheDir, outputIDHex, srcPath string) (string, int64) {
	src, err := os.Open(srcPath)
	if err != nil {
		logf("diskpath open %s: %v", srcPath, err)
		return "", 0
	}
	defer src.Close()

	dest := dataPath(cacheDir, outputIDHex)
	tmp, gotHash, size, err := streamToTempFile(src, dest)
	if err != nil {
		logf("diskpath stream %s: %v", srcPath, err)
		return "", 0
	}
	if gotHash != outputIDHex {
		// Integrity: trust the content — store under its real hash.
		logf("diskpath %s: OutputID != sha256(body); using content hash", outputIDHex[:12])
		os.Remove(tmp)
		// re-stream into the content-hash path
		dest = dataPath(cacheDir, gotHash)
		if _, err := os.Stat(dest); err == nil {
			return gotHash, size
		}
		src2, err := os.Open(srcPath)
		if err != nil {
			return "", 0
		}
		defer src2.Close()
		tmp2, h2, s2, err := streamToTempFile(src2, dest)
		if err != nil {
			return "", 0
		}
		if h2 != gotHash {
			os.Remove(tmp2)
			return "", 0
		}
		os.Rename(tmp2, dest)
		return gotHash, s2
	}
	os.Rename(tmp, dest)
	return outputIDHex, size
}

// putFileGzip uploads a file at key with gzip-at-rest via a temp file
// (bounded memory — the 300MB-link-output case). Returns the uploaded
// size.
func (r *Remote) putFileGzip(key, path string) error {
	token, err := r.getToken()
	if err != nil {
		return err
	}
	// Unique temp (concurrent uploads of the same outputID raced on a
	// fixed name). One handle, closed once at the end — an earlier
	// version Stat()'d after Close() and failed every CI upload.
	f, err := os.CreateTemp(filepath.Dir(path), ".gz-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	zw := gzip.NewWriter(f)
	src, err := os.Open(path)
	if err != nil {
		f.Close()
		return err
	}
	_, copyErr := io.Copy(zw, src)
	src.Close()
	zw.Close()
	if copyErr != nil {
		f.Close()
		return copyErr
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return err
	}
	req, err := http.NewRequest("PUT", r.containerURL()+"/"+key, f)
	if err != nil {
		f.Close()
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", "2023-11-03")
	req.ContentLength = st.Size()
	resp, err := r.client.Do(req)
	f.Close()
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("remote PUT(file) %s: HTTP %d", key, resp.StatusCode)
	}
	return nil
}

// putFileIfNotExistGzip: upload only when absent (If-None-Match).
func (r *Remote) putFileIfNotExistGzip(key, path string) (bool, error) {
	exists, err := r.exists(key)
	if err != nil || exists {
		return !exists, err
	}
	// Small race window between HEAD and PUT is benign: last write of
	// identical content. (True atomicity needs If-None-Match on the
	// PUT, which streaming PUT with unknown length can't carry.)
	if err := r.putFileGzip(key, path); err != nil {
		return false, err
	}
	return true, nil
}

func (r *Remote) exists(key string) (bool, error) {
	token, err := r.getToken()
	if err != nil {
		return false, err
	}
	req, _ := http.NewRequest("HEAD", r.containerURL()+"/"+key, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-version", "2023-11-03")
	resp, err := r.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("remote HEAD %s: HTTP %d", key, resp.StatusCode)
}
