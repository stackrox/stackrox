package proxy

import (
	"context"
	"io/ioutil"
	"net"
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
	log            = logging.LoggerForModule()
	proxyHandler   *reloadProxyConfigHandler
	proxyTransport *http.Transport
	proxyOnce      sync.Once
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
	r.updateEnvNoLock()
}

func (r *reloadProxyConfigHandler) updateEnvNoLock() {
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
	if os.IsNotExist(err) {
		return
	}
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
	r.updateEnvNoLock()
}

func initHandler() {
	proxyOnce.Do(func() {
		opts := k8scfgwatch.Options{
			Force:    true,
			Interval: proxyReloadInterval,
		}
		proxyHandler = &reloadProxyConfigHandler{}
		_ = k8scfgwatch.WatchConfigMountDir(context.Background(), proxyConfigPath, k8scfgwatch.DeduplicateWatchErrors(proxyHandler), opts)
		trans, _ := http.DefaultTransport.(*http.Transport)
		if trans != nil {
			proxyTransport = trans.Clone()
			proxyTransport.Proxy = proxyHandler.proxy
		} else {
			// fallback copied from go http/transport.go, circa 1.13.1.
			proxyTransport = &http.Transport{
				Proxy: proxyHandler.proxy,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			}
		}
	})
}

// FromConfig returns an function suitable for use as a Proxy field in an *http.Transport instance that will always
// use the latest configured proxy setting.
func FromConfig() func(*http.Request) (*url.URL, error) {
	initHandler()
	return proxyHandler.proxy
}

// EnableProxyEnvironmentSetting enables the behavior of mutating the current process's environment to always have
// the most up to date setting. This is specifically useful when running child processes that need to access the
// environment.
// Note: setting this flag to false after it was set to true will not clear the environment.
func EnableProxyEnvironmentSetting(enable bool) {
	initHandler()
	if enable && (os.Getenv("https_proxy") != "" || os.Getenv("http_proxy") != "") {
		log.Warn("Not enabling proxy environment override because the environment is already configured")
		return
	}
	proxyHandler.enableProxySetting(enable)
}

// RoundTripper returns something very similar to http.DefaultTransport, but with the Proxy setting changed to use
// the configuration supported by this package.
func RoundTripper() http.RoundTripper {
	initHandler()
	return proxyTransport
}
