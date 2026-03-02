package datastore

import (
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	minUpdateInterval  = 1 * time.Hour
	defaultManifestURL = "https://storage.googleapis.com/rox-public-key-test-20260203/manifest.json"
)

type publicKey struct {
	name            string
	publicKeyPemEnc string
}

type publicKeyManifest struct {
	Keys []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"keys"`
}

type updater struct {
	client      *http.Client
	interval    time.Duration
	manifestURL string
	once        sync.Once
	stopSig     concurrency.Signal
}

func newUpdater() *updater {
	interval := env.RedHatSigningKeyUpdateInterval.DurationSetting()
	if interval < minUpdateInterval {
		log.Warnf("ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL is too short, setting to the minimum duration (%v)", minUpdateInterval)
		interval = minUpdateInterval
	}

	return &updater{
		client: &http.Client{
			Transport: proxy.RoundTripper(),
			Timeout:   5 * time.Minute,
		},
		interval:    interval,
		manifestURL: defaultManifestURL,
		stopSig:     concurrency.NewSignal(),
	}
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
	log.Infof("Starting to update the default Red Hat signature integration every %v", u.interval)

	// Run an initial update, to handle cases where the key was rotated but the backed-in key (pkg/signatures/release-key-3.pub.txt)
	// is still the old one. Without this, the default Red Hat signature integration would have an outdated key during
	// the first `u.interval`.
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
		log.Errorf("Failed to update the default Red Hat signature integration: %v", err)
	}
}

func (u *updater) update() error {
	publicKeys, err := u.fetchPublicKeysFromManifest(u.manifestURL)
	if err != nil {
		return errors.Wrapf(err, "fetching public keys from manifest (manifest URL: %s)", u.manifestURL)
	}
	if len(publicKeys) == 0 {
		return errors.Errorf("no valid public keys could be fetched (manifest URL: %s)", u.manifestURL)
	}

	if err = u.updateKeysInSignatureIntegration(publicKeys); err != nil {
		return errors.Wrap(err, "updating keys in signature integration")
	}

	log.Debugf("Updated %d public keys in the default Red Hat signature integration", len(publicKeys))

	return nil
}

func (u *updater) fetchPublicKeysFromManifest(manifestURL string) ([]publicKey, error) {
	publicKeys := []publicKey{}

	manifest, err := u.fetchManifest(manifestURL)
	if err != nil {
		return publicKeys, err
	}

	for _, keyRef := range manifest.Keys {
		keyURL, err := resolveKeyURL(manifestURL, keyRef.URL)
		if err != nil {
			log.Warnf("Failed to resolve key URL %q: %v", keyRef.URL, err)
			continue
		}
		key, err := u.fetchPublicKey(keyRef.Name, keyURL)
		if err != nil {
			log.Warnf("Failed to fetch public key %s from %s: %v", keyRef.Name, keyURL, err)
			continue
		}
		publicKeys = append(publicKeys, key)
	}

	log.Debugf("Fetched %d public keys from manifest at %s", len(publicKeys), manifestURL)

	return publicKeys, nil

}

func (u *updater) fetchPublicKey(name, url string) (publicKey, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return publicKey{}, errors.Wrap(err, "constructing request")
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return publicKey{}, errors.Wrap(err, "executing request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return publicKey{}, errors.Errorf("HTTP response code was %d", resp.StatusCode)
	}

	keyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return publicKey{}, errors.Wrap(err, "reading response body")
	}

	publicKeyPemEnc := string(keyBytes)
	if err = validatePublicKey(publicKeyPemEnc); err != nil {
		return publicKey{}, errors.Wrapf(err, "validating public key from %s", url)
	}

	return publicKey{
		name:            name,
		publicKeyPemEnc: publicKeyPemEnc,
	}, nil
}

func (u *updater) fetchManifest(manifestURL string) (publicKeyManifest, error) {
	req, err := http.NewRequest(http.MethodGet, manifestURL, nil)
	if err != nil {
		return publicKeyManifest{}, errors.Wrap(err, "constructing request")
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return publicKeyManifest{}, errors.Wrap(err, "executing request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return publicKeyManifest{}, errors.Errorf("HTTP response code was %d", resp.StatusCode)
	}

	manifestBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return publicKeyManifest{}, errors.Wrap(err, "reading response body")
	}

	var manifest publicKeyManifest
	if err = json.Unmarshal(manifestBytes, &manifest); err != nil {
		return publicKeyManifest{}, errors.Wrap(err, "unmarshalling manifest")
	}

	log.Debugf("Fetched manifest at %s: %+v", manifestURL, manifest)

	return manifest, nil
}

func (u *updater) updateKeysInSignatureIntegration(publicKeys []publicKey) error {
	log.Debugf("Updating Red Hat signing keys in the default Red Hat signature integration")

	integration := signatures.DefaultRedHatSignatureIntegration.CloneVT()
	integration.Cosign.PublicKeys = make([]*storage.CosignPublicKeyVerification_PublicKey, 0, len(publicKeys))
	for _, key := range publicKeys {
		integration.Cosign.PublicKeys = append(integration.Cosign.PublicKeys, &storage.CosignPublicKeyVerification_PublicKey{
			Name:            key.name,
			PublicKeyPemEnc: key.publicKeyPemEnc,
		})
	}

	return upsertDefaultRedHatSignatureIntegration(siStore, integration)
}

// resolveKeyURL returns the full URL for a manifest key entry. If entry is already absolute (http(s)://), it is returned as-is; otherwise it is resolved relative to the manifest URL's directory.
// It returns an error if the resulting URL does not point to a file (e.g. path ends with /).
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
		// Strip the last path segment (e.g. manifest.json) to get the directory.
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

func validatePublicKey(key string) error {
	keyBlock, rest := pem.Decode([]byte(key))
	if !signatures.IsValidPublicKeyPEMBlock(keyBlock, rest) {
		return errors.New("failed to decode PEM block containing public key")
	}
	return nil
}
