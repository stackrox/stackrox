package manager

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	path = "/run/secrets/stackrox.io/integrations/gcs_backup/"
	file = "proto"
)

var (
	elevatedCtx = sac.WithAllAccess(context.Background())
)

// StartBackupFromConfigManager starts a goroutine which watches for changes in a pre-configured external backup
func StartBackupFromConfigManager(mgr Manager) {
	if !features.IntegrationsAsConfig.Enabled() {
		return
	}
	wh := &watchHandler{
		mgr:  mgr,
		file: file,
	}

	watchOpts := k8scfgwatch.Options{
		Interval: 5 * time.Second,
		Force:    true,
	}

	_ = k8scfgwatch.WatchConfigMountDir(elevatedCtx, path, k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)
}

type watchHandler struct {
	mgr  Manager
	id   string
	file string
}

func getConfig(filename string) (*v1.IntegrationAsConfiguration, error) {
	yamlBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "reading from file")
	}
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return nil, errors.Wrap(err, "converting to json")
	}
	var config v1.IntegrationAsConfiguration
	err = jsonutil.JSONBytesToProto(jsonBytes, &config)
	if err != nil {
		return nil, errors.Wrapf(err, "error unmarshaling proto from secret %q", filename)
	}
	return &config, nil
}

func (w *watchHandler) OnChange(dir string) (interface{}, error) {
	fullPath := filepath.Join(dir, w.file)
	config, err := getConfig(fullPath)
	if err != nil {
		return nil, err
	}

	backupConfig := config.GetExternalBackup()
	if backupConfig == nil {
		return nil, errors.Errorf("No external backup configuration in secret %q", fullPath)
	}

	return backupConfig, nil
}

func (w *watchHandler) OnStableUpdate(val interface{}, err error) {
	if err != nil {
		log.Error(err)
		if w.id != "" {
			w.mgr.Remove(elevatedCtx, w.id)
		}
		return
	}
	if val == nil {
		// This should never happen
		w.mgr.Remove(elevatedCtx, w.id)
		return
	}

	// val is guaranteed to be an ExternalBackup.  It is the return value from OnChange.
	config := val.(*storage.ExternalBackup)
	if w.id != "" {
		config.Id = w.id
	}
	upsertErr := w.mgr.Upsert(elevatedCtx, config)
	if upsertErr != nil {
		log.Errorf("error adding new external backup from config: %v", upsertErr)
		return
	}
	w.id = config.Id
}

func (w *watchHandler) OnWatchError(_ error) {
	if w.id != "" {
		w.mgr.Remove(elevatedCtx, w.id)
	}
}
