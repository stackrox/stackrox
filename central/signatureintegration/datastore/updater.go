package datastore

import (
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// updater periodically updates Red Hat's software signing key in the default Red Hat signature integration.
type updater struct {
	client      *http.Client
	interval    time.Duration
	once        sync.Once
	stopSig     concurrency.Signal
	url         string
	previousKey string
}

func newUpdater() *updater {
	return &updater{
		client: &http.Client{
			Transport: proxy.RoundTripper(),
			Timeout:   5 * time.Minute,
		},
		interval:    env.RedHatSigningKeyUpdateInterval.DurationSetting(),
		previousKey: signatures.ReleaseKey3PublicKey,
		stopSig:     concurrency.NewSignal(),
		url:         env.RedHatSigningKeyBucketURL.Setting(),
	}
}

// Stop stops the updater.
func (u *updater) Stop() {
	u.stopSig.Signal()
}

// Start starts the updater.
// The updater is only started once.
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
	key, err := u.fetchPublicKey()
	if err != nil {
		return err
	}

	if key == u.previousKey {
		log.Infof("Skipping update of default Red Hat signature integration because the key has not changed")
		return nil
	}

	if err = u.updateKeyInSignatureIntegration(key); err != nil {
		return err
	}

	u.previousKey = key

	return nil
}

func (u *updater) fetchPublicKey() (string, error) {
	log.Debugf("Sending GET request to %s", u.url)

	req, err := http.NewRequest(http.MethodGet, u.url, nil)
	if err != nil {
		return "", errors.Wrap(err, "constructing request")
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "executing request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("HTTP response code was %d", resp.StatusCode)
	}

	keyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "reading response body")
	}

	return string(keyBytes), nil
}

func (u *updater) updateKeyInSignatureIntegration(key string) error {
	log.Debugf("Updating Red Hat signing key in the default Red Hat signature integration to %s", key)

	integration := signatures.DefaultRedHatSignatureIntegration
	integration.Cosign.PublicKeys[0].PublicKeyPemEnc = key

	return upsertDefaultRedHatSignatureIntegration(siStore, integration)
}
