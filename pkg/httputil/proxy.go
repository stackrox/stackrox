package httputil

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/ghodss/yaml"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	proxyConfigPath     = "/run/secrets/stackrox.io/proxy-config"
	proxyConfigFile     = "config.yaml"
	proxyReloadInterval = 5 * time.Second
)

var (
	log          = logging.LoggerForModule()
	proxyHandler *reloadProxyConfigHandler
	proxyOnce    sync.Once
)

type proxyConfig struct {
	ProxyURL string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (pc proxyConfig) toURL() (*url.URL, error) {
	if pc.ProxyURL == "" {
		return nil, nil
	}
	u, err := url.Parse(pc.ProxyURL)
	if err != nil {
		return nil, err
	}
	if pc.Username != "" {
		u.User = url.User(pc.Username)
		if pc.Password != "" {
			u.User = url.UserPassword(pc.Username, pc.Password)
		}
	}
	return u, nil
}

type reloadProxyConfigHandler struct {
	mutex    sync.Mutex
	proxyURL *url.URL
	setEnv   bool
}

func (r *reloadProxyConfigHandler) OnChange(dir string) (interface{}, error) {
	var out proxyConfig
	configBytes, err := ioutil.ReadFile(filepath.Join(dir, proxyConfigFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	err = yaml.Unmarshal(configBytes, &out)
	if err != nil {
		return nil, err
	}
	return out.toURL()
}

func (r *reloadProxyConfigHandler) OnStableUpdate(val interface{}, err error) {
	if err != nil {
		log.Errorf("Error reading proxy config file: %v", err)
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.proxyURL, _ = val.(*url.URL)
	if !r.setEnv {
		return
	}
	if r.proxyURL == nil {
		_ = os.Unsetenv("https_proxy")
		_ = os.Unsetenv("http_proxy")
	} else {
		proxyString := r.proxyURL.String()
		_ = os.Setenv("https_proxy", proxyString)
		_ = os.Setenv("http_proxy", proxyString)
	}
}

func (r *reloadProxyConfigHandler) OnWatchError(err error) {
	log.Errorf("Error watching for proxy config changes: %v", err)
}

func (r *reloadProxyConfigHandler) proxy(_ *http.Request) (*url.URL, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.proxyURL, nil
}

func (r *reloadProxyConfigHandler) enableProxySetting(enable bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.setEnv = enable
}

func initHandler() {
	proxyOnce.Do(func() {
		opts := k8scfgwatch.Options{
			Force:    true,
			Interval: proxyReloadInterval,
		}
		proxyHandler = &reloadProxyConfigHandler{}
		_ = k8scfgwatch.WatchConfigMountDir(context.Background(), proxyConfigPath, proxyHandler, opts)
	})
}

// ProxyFunc returns an function suitable for use as a Proxy field in an *http.Transport instance that will always
// use the latest configured proxy setting.
func ProxyFunc() func(*http.Request) (*url.URL, error) {
	initHandler()
	return proxyHandler.proxy
}

// EnableProxyEnvironmentSetting enables the behavior of mutating the current process's environment to always have
// the most up to date setting. This is specifically useful when
func EnableProxyEnvironmentSetting(enable bool) {
	initHandler()
	if enable && (os.Getenv("https_proxy") != "" || os.Getenv("http_proxy") != "") {
		log.Warn("Not enabling proxy environment override because the environment is already configured")
		return
	}
	proxyHandler.enableProxySetting(enable)
}
