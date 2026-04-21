package datastore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

type manifest struct {
	Keys []manifestKey `json:"keys"`
}

type manifestKey struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type keyRef struct {
	name string
	url  string
}

type updater struct {
	client      *http.Client
	interval    time.Duration
	manifestURL string
	targetDir   string
	once        sync.Once
	stopSig     concurrency.Signal
}

func newUpdater(client *http.Client, manifestURL, targetDir string, interval time.Duration) (*updater, error) {
	if client == nil {
		return nil, errors.New("http client must be provided")
	}
	if interval <= 0 {
		return nil, errors.New("update interval must be positive")
	}

	return &updater{
		client:      client,
		interval:    interval,
		manifestURL: manifestURL,
		targetDir:   targetDir,
		stopSig:     concurrency.NewSignal(),
	}, nil
}

func (u *updater) Stop() {
	u.stopSig.Signal()
}

func (u *updater) Start() {
	u.once.Do(func() {
		go u.runForever()
	})
}

func (u *updater) runForever() {
	log.Infof("Starting Red Hat key file updater every %v", u.interval)
	u.doUpdate()

	t := time.NewTimer(u.interval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			u.doUpdate()
			t.Reset(u.interval)
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *updater) doUpdate() {
	if err := u.update(); err != nil {
		log.Errorf("Failed to download Red Hat key files from manifest %q into %q: %v", u.manifestURL, u.targetDir, err)
	}
}

func (u *updater) update() error {
	mf, err := u.downloadManifest(u.manifestURL)
	if err != nil {
		return errors.Wrapf(err, "downloading manifest from URL %q", u.manifestURL)
	}

	keyRefs, err := resolveKeyRefsFromManifest(u.manifestURL, mf)
	if err != nil {
		return errors.Wrap(err, "resolving key references from manifest")
	}
	if err := os.MkdirAll(u.targetDir, 0o755); err != nil {
		return errors.Wrapf(err, "creating target directory %q", u.targetDir)
	}
	stagingDir, err := os.MkdirTemp(filepath.Dir(u.targetDir), ".redhat-keys-staging-*")
	if err != nil {
		return errors.Wrapf(err, "creating staging directory for target %q", u.targetDir)
	}
	defer func() {
		_ = os.RemoveAll(stagingDir)
	}()

	n, err := u.downloadKeys(keyRefs, stagingDir)
	if err != nil {
		return errors.Wrap(err, "downloading keys to staging")
	}
	if err := replaceDirectoryContents(u.targetDir, stagingDir); err != nil {
		return errors.Wrapf(err, "replacing target directory %q contents", u.targetDir)
	}

	log.Infof("Successfully updated Red Hat signing keys from %q: %d keys written to %q", u.manifestURL, n, u.targetDir)
	return nil
}

// resolveKeyRefsFromManifest resolves the key references from a manifest.
func resolveKeyRefsFromManifest(manifestURL string, mf manifest) ([]keyRef, error) {
	if len(mf.Keys) == 0 {
		return nil, errors.Errorf("manifest at %q does not contain any files", manifestURL)
	}

	keyRefs := make([]keyRef, 0, len(mf.Keys))
	for _, key := range mf.Keys {
		name := strings.TrimSpace(key.Name)
		if name == "" {
			return nil, errors.New("manifest entry has empty name")
		}
		if strings.ContainsAny(name, "/\\") {
			return nil, errors.Errorf("manifest entry name %q contains path separator", name)
		}
		resolvedURL, err := resolveKeyURL(manifestURL, key.URL)
		if err != nil {
			return nil, errors.Wrapf(err, "resolving URL %q", key.URL)
		}
		keyRefs = append(keyRefs, keyRef{
			name: name,
			url:  resolvedURL,
		})
	}
	return keyRefs, nil
}

func (u *updater) downloadKeys(keys []keyRef, stagingDir string) (int, error) {
	successes := 0
	failures := 0

	cleanStagingDir := filepath.Clean(stagingDir) + string(os.PathSeparator)
	for _, key := range keys {
		destination := filepath.Join(stagingDir, key.name)
		if !strings.HasPrefix(filepath.Clean(destination)+string(os.PathSeparator), cleanStagingDir) {
			failures++
			log.Warnf("Skipping manifest entry %q (URL %q): resolved path escapes staging directory", key.name, key.url)
			continue
		}
		if err := u.downloadFile(key.url, destination); err != nil {
			failures++
			log.Warnf("Skipping manifest entry %q (URL %q): %v", key.name, key.url, err)
			continue
		}
		successes++
	}

	if successes == 0 {
		return 0, errors.New("failed to download any keys")
	}
	if failures > 0 {
		log.Warnf("Downloaded %d keys, skipped %d entries", successes, failures)
	}

	return successes, nil
}

func (u *updater) downloadFile(url string, destination string) error {
	contents, err := u.downloadBytes(url)
	if err != nil {
		return errors.Wrapf(err, "failed to download file from URL %q", url)
	}

	if err := os.WriteFile(destination, contents, 0o644); err != nil {
		_ = os.Remove(destination)
		return errors.Wrapf(err, "failed to write file %q", destination)
	}

	return nil
}

func replaceDirectoryContents(targetDir, sourceDir string) error {
	backupDir := targetDir + ".backup-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.Rename(targetDir, backupDir); err != nil {
		return errors.Wrap(err, "moving current target directory to backup")
	}
	if err := os.Rename(sourceDir, targetDir); err != nil {
		rollbackErr := os.Rename(backupDir, targetDir)
		if rollbackErr != nil {
			return errors.Wrapf(err, "moving staged directory into target failed and rollback failed: %v", rollbackErr)
		}
		return errors.Wrap(err, "moving staged directory into target")
	}
	if err := os.RemoveAll(backupDir); err != nil {
		log.Warnf("Failed to remove backup directory %q: %v", backupDir, err)
	}

	return nil
}

// maxResponseBodySize is the maximum size of a response body (manifest or key file) we will read.
const maxResponseBodySize = 5 * 1024 * 1024 // 5 MB

func (u *updater) downloadBytes(rawURL string) ([]byte, error) {
	reqCtx, cancel := u.newRequestContext()
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "constructing request")
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "executing request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("HTTP %d for %q", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize+1))
	if err != nil {
		return nil, errors.Wrap(err, "reading response body")
	}
	if len(body) > maxResponseBodySize {
		return nil, errors.Errorf("response body from %q exceeds maximum size of %d bytes", rawURL, maxResponseBodySize)
	}

	return body, nil
}

const requestTimeout = 60 * time.Second

func (u *updater) newRequestContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	concurrency.CancelContextOnSignal(ctx, cancel, &u.stopSig)
	return ctx, cancel
}

func (u *updater) downloadManifest(manifestURL string) (manifest, error) {
	manifestBytes, err := u.downloadBytes(manifestURL)
	if err != nil {
		return manifest{}, err
	}

	var parsedManifest manifest
	if err = json.Unmarshal(manifestBytes, &parsedManifest); err != nil {
		return manifest{}, errors.Wrap(err, "unmarshalling manifest")
	}

	return parsedManifest, nil
}

// resolveKeyURL resolves a key URL relative to a manifest URL.
// The key URL can be absolute or relative.
// If it is relative, it is resolved relative to the manifest URL.
// If it is absolute, it is returned as is.
func resolveKeyURL(manifestURL, keyURL string) (string, error) {
	keyURL = strings.TrimSpace(keyURL)

	var resolved string
	if strings.HasPrefix(keyURL, "http://") || strings.HasPrefix(keyURL, "https://") {
		resolved = keyURL
	} else {
		base, err := url.Parse(manifestURL)
		if err != nil {
			return "", errors.Wrap(err, "parsing manifest URL")
		}
		base.Path = strings.TrimSuffix(base.Path, "/")
		if idx := strings.LastIndex(base.Path, "/"); idx >= 0 {
			base.Path = base.Path[:idx+1]
		}

		ref, err := url.Parse(keyURL)
		if err != nil {
			return "", errors.Wrap(err, "parsing key entry")
		}
		resolved = base.ResolveReference(ref).String()
	}

	parsed, err := url.Parse(resolved)
	if err != nil {
		return "", errors.Wrap(err, "parsing resolved URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.Errorf("unsupported URL scheme %q in %s", parsed.Scheme, resolved)
	}
	if strings.HasSuffix(parsed.Path, "/") || parsed.Path == "" {
		return "", errors.Errorf("URL must point to a file, not a directory: %s", resolved)
	}

	return resolved, nil
}
