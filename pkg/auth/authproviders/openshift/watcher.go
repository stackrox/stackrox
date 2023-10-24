package openshift

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()

	_ k8scfgwatch.Handler = (*handler)(nil)
)

type notifyCertPoolUpdate = func()

func watchCertPool(n notifyCertPoolUpdate) {
	opts := k8scfgwatch.Options{
		Interval: 5 * time.Second,
		Force:    true,
	}

	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), path.Dir(serviceOperatorCAPath),
		k8scfgwatch.DeduplicateWatchErrors(&handler{readCAs: internalCAs, onCertPoolUpdate: n}), opts)
	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), path.Dir(injectedCAPath),
		k8scfgwatch.DeduplicateWatchErrors(&handler{readCAs: injectedCAs, onCertPoolUpdate: n}), opts)
}

type handler struct {
	onCertPoolUpdate func()
	readCAs          func() ([][]byte, error)
}

func (h *handler) OnChange(_ string) (interface{}, error) {
	return h.readCAs()
}

func (h *handler) OnStableUpdate(val interface{}, err error) {
	// Ignore errors and nil values.
	if err != nil || val == nil {
		return
	}

	// Expect always a [][]byte.
	caBytes := val.([][]byte)
	if caBytes == nil {
		log.Info("No updated CA bytes found, using the default system CA cert pool.")
		return
	}

	log.Info("Found an update to the root CAs for Openshift auth providers. Updating the providers.")
	h.onCertPoolUpdate()
}

func (h *handler) OnWatchError(err error) {
	if !os.IsNotExist(err) {
		log.Errorw("Failed watching CAs.",
			logging.Err(err))
	}
}

func internalCAs() ([][]byte, error) {
	var caBytes [][]byte
	ca, exists, err := readCA(serviceOperatorCAPath)
	if err != nil {
		return nil, err
	}
	if exists {
		caBytes = append(caBytes, ca)
	}
	ca, exists, err = readCA(internalServicesCAPath)
	if err != nil {
		return nil, err
	}
	if exists {
		caBytes = append(caBytes, ca)
	}
	return caBytes, nil
}

func injectedCAs() ([][]byte, error) {
	caBytes, exists, err := readCA(injectedCAPath)
	if err != nil {
		return nil, err
	}
	if exists {
		return [][]byte{caBytes}, nil
	}
	return nil, nil
}

func readCA(file string) ([]byte, bool, error) {
	caBytes, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		log.Errorw("Reading CA file", logging.Err(err), logging.String("file", file))
		return nil, false, err
	}

	return caBytes, true, nil
}
