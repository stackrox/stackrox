package datastore

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const defaultUpdateInterval = 4 * time.Hour

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
		interval = defaultUpdateInterval
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
	manifest, err := u.downloadManifest(u.manifestURL)
	if err != nil {
		return errors.Wrapf(err, "downloading manifest from URL %q", u.manifestURL)
	}

	keyRefs, err := resolveKeyRefsFromManifest(u.manifestURL, manifest)
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

	if err := u.downloadKeys(keyRefs, stagingDir); err != nil {
		return errors.Wrap(err, "downloading keys to staging")
	}
	if err := replaceDirectoryContents(u.targetDir, stagingDir); err != nil {
		return errors.Wrapf(err, "replacing target directory %q contents", u.targetDir)
	}

	return nil
}

// resolveKeyRefsFromManifest resolves the key references from a manifest.
func resolveKeyRefsFromManifest(manifestURL string, manifest manifest) ([]keyRef, error) {
	if len(manifest.Keys) == 0 {
		return nil, errors.Errorf("manifest at %q does not contain any files", manifestURL)
	}

	keyRefs := make([]keyRef, 0, len(manifest.Keys))
	for _, key := range manifest.Keys {
		resolvedURL, err := resolveKeyURL(manifestURL, key.URL)
		if err != nil {
			return nil, errors.Wrapf(err, "resolving URL %q", key.URL)
		}
		keyRefs = append(keyRefs, keyRef{
			name: key.Name,
			url:  resolvedURL,
		})
	}
	return keyRefs, nil
}

func (u *updater) downloadKeys(keys []keyRef, stagingDir string) error {
	successes := 0
	failures := 0

	for _, key := range keys {
		destination := filepath.Join(stagingDir, key.name)
		if err := u.downloadFile(key.url, destination); err != nil {
			failures++
			log.Warnf("Skipping manifest entry %q: %v", key.url, err)
			continue
		}
		successes++
	}

	if successes == 0 {
		return errors.New("failed to download any keys")
	}
	if failures > 0 {
		log.Warnf("Downloaded %d keys to staging %q, skipped %d entries", successes, stagingDir, failures)
	} else {
		log.Debugf("Downloaded %d keys to staging %q", successes, stagingDir)
	}

	return nil
}

func (u *updater) downloadFile(url string, destination string) error {
	contents, err := u.downloadBytes(url)
	if err != nil {
		return errors.Wrapf(err, "failed to download file from URL %q", url)
	}

	if err := os.WriteFile(destination, contents, 0o600); err != nil {
		_ = os.Remove(destination)
		return errors.Wrapf(err, "failed to write file %q", destination)
	}

	return nil
}

func replaceDirectoryContents(targetDir, sourceDir string) error {
	if err := clearDirectory(targetDir); err != nil {
		return errors.Wrap(err, "clearing target directory")
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return errors.Wrap(err, "reading source directory")
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(sourceDir, entry.Name())
		targetPath := filepath.Join(targetDir, entry.Name())
		if err := os.Rename(sourcePath, targetPath); err != nil {
			return errors.Wrapf(err, "moving %q to target directory", entry.Name())
		}
	}

	return nil
}

func clearDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return errors.Wrap(err, "reading directory entries")
	}
	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return errors.Wrapf(err, "removing %q", entryPath)
		}
	}

	return nil
}

func (u *updater) downloadBytes(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "constructing request")
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "executing request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("HTTP response code was %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading response body")
	}

	return body, nil
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
	if strings.HasSuffix(parsed.Path, "/") || parsed.Path == "" {
		return "", errors.Errorf("URL must point to a file, not a directory: %s", resolved)
	}

	return resolved, nil
}
