package jwt

import (
	"context"
	"crypto/rsa"
	"encoding/pem"
	"time"

	"github.com/google/certificate-transparency-go/x509"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
)

const (
	watchInterval = 1 * time.Minute
)

var (
	_ k8scfgwatch.Handler = (*handler)(nil) // compile time check that the handler implements the interface
)

type loadKeyFunc func(dir string) (*rsa.PrivateKey, error)
type updateKeyFunc func(key *rsa.PrivateKey)

// WatchKeyDir starts watching the directory containing certificates
func WatchKeyDir(dir string, loadKey loadKeyFunc, updateKey updateKeyFunc) {
	wh := &handler{
		dir:       dir,
		loadKey:   loadKey,
		updateKey: updateKey,
	}

	watchOpts := k8scfgwatch.Options{
		Interval: watchInterval,
		Force:    true,
	}
	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), dir, k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)
}

type handler struct {
	dir       string
	loadKey   loadKeyFunc
	updateKey updateKeyFunc
}

func (h *handler) OnChange(dir string) (interface{}, error) {
	return h.loadKey(dir)
}

func (h *handler) OnStableUpdate(val interface{}, err error) {
	var key *rsa.PrivateKey
	if err != nil {
		log.Errorf("Error reading JWT private key: %v. Skipping incoming update. Watch dir: %q", err, h.dir)
		return
	} else {
		key, _ = val.(*rsa.PrivateKey)
		if key == nil {
			log.Infof("No private key found. Using previous value. Watch dir: %q", h.dir)
			return
		}
	}
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(key)
	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	log.Infof("Here's the key: %s", string(pemBlock))
	h.updateKey(key)
}

func (h *handler) OnWatchError(err error) {
	log.Errorf("Error watching TLS certificate directory %q: %v. Not updating TLS certificates!", h.dir, err)
}
