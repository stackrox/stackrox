package proxy

import (
	"context"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
)

type reloadProxyConfigHandler struct {
	cfgFile string
	setEnv  bool
}

func (r *reloadProxyConfigHandler) OnChange(dir string) (interface{}, error) {
	var out proxyConfig
	configBytes, err := os.ReadFile(filepath.Join(dir, r.cfgFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	err = yaml.Unmarshal(configBytes, &out)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling proxy config YAML")
	}
	if err := out.Validate(); err != nil {
		return nil, errors.Wrap(err, "validating updated proxy configuration")
	}

	return out.Compile(initialEnvCfg), nil
}

func (r *reloadProxyConfigHandler) OnStableUpdate(val interface{}, err error) {
	if err != nil {
		log.Errorf("Error reading proxy config file: %v. Not modifying proxy config", err)
		return
	}

	cfg, _ := val.(*compiledConfig)
	// If we have a `nil` value and a `nil` error, this means the config file doesn't exist or has been removed.
	// In this case switch back to using the default, environment-based config (which is accomplished by storing
	// a `nil` config in `globalProxyConfig`).
	globalProxyConfig.Store(cfg)
	if cfg == nil {
		cfg = defaultProxyConfig
	}
	if r.setEnv {
		cfg.SetEnv()
	}
}

func (r *reloadProxyConfigHandler) OnWatchError(err error) {
	if os.IsNotExist(err) {
		r.OnStableUpdate(nil, nil)
		return
	}
	log.Errorf("Error watching for proxy config changes: %v", err)
}

// WatchProxyConfig triggers a proxy config watcher goroutine. Important: it is the callers responsibility to ensure
// that not two of these goroutines ever run at the same time!
func WatchProxyConfig(ctx context.Context, cfgDir, cfgFile string, updateEnv bool) {
	opts := k8scfgwatch.Options{
		Force:    true,
		Interval: proxyReloadInterval,
	}

	handler := &reloadProxyConfigHandler{
		cfgFile: cfgFile,
		setEnv:  updateEnv,
	}
	_ = k8scfgwatch.WatchConfigMountDir(ctx, cfgDir, k8scfgwatch.DeduplicateWatchErrors(handler), opts)
}
