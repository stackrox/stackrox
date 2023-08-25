package proxy

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sliceutils"
	"golang.org/x/net/http/httpproxy"
)

var (
	defaultExcludes = []string{
		"*.stackrox",
		"*.stackrox.svc",
		"localhost",
		"localhost.localdomain",
		"127.0.0.0/8",
		"::1",
		"*.local",
	}
)

func urlToStringOrEmpty(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.String()
}

type proxyEndpointConfig struct {
	ProxyURL string  `json:"url"`
	Username string  `json:"username"`
	Password *string `json:"password"`
}

func (c *proxyEndpointConfig) toURL() (*url.URL, error) {
	if c.ProxyURL == "" {
		return nil, nil
	}

	u, err := url.Parse(c.ProxyURL)
	if err != nil {
		return nil, errors.Wrap(err, "malformed proxy URL")
	}

	if c.Username != "" {
		if u.User != nil {
			return nil, errors.New("username and password must not be set if the URL contains `user[:password]@`")
		}
		if c.Password == nil {
			u.User = url.User(c.Username)
		} else {
			u.User = url.UserPassword(c.Username, *c.Password)
		}
	} else if c.Password != nil {
		return nil, errors.New("password set in config requires a non-empty username")
	}

	return u, nil
}

func (c *proxyEndpointConfig) Validate() error {
	_, err := c.toURL()
	if err != nil {
		return err
	}

	if c.ProxyURL == "" && (c.Username != "" || c.Password != nil) {
		return errors.New("username/password may only be set if url is set")
	}
	return nil
}

type proxyConfig struct {
	proxyEndpointConfig
	HTTP  proxyEndpointConfig `json:"http"`
	HTTPS proxyEndpointConfig `json:"https"`

	Excludes            []string `json:"excludes"`
	OmitDefaultExcludes bool     `json:"omitDefaultExcludes"`
}

func (c *proxyConfig) Validate() error {
	errs := errorhelpers.NewErrorList("proxy configuration failed validation")
	if err := c.proxyEndpointConfig.Validate(); err != nil {
		errs.AddWrap(err, "default proxy config")
	}
	if err := c.HTTP.Validate(); err != nil {
		errs.AddWrap(err, "HTTP proxy config")
	}
	if err := c.HTTPS.Validate(); err != nil {
		errs.AddWrap(err, "HTTPS proxy config")
	}
	return errs.ToError()
}

type compiledConfig struct {
	httpFunc, httpsFunc, otherFunc func(*url.URL) (*url.URL, error)

	envVars map[string]string
}

func (c *compiledConfig) ProxyURL(req *http.Request) (*url.URL, error) {
	switch u := req.URL; u.Scheme {
	case "http":
		return c.httpFunc(u)
	case "https":
		return c.httpsFunc(u)
	default:
		modifiedURL := *u
		modifiedURL.Scheme = "http" // use http scheme for simplicity
		return c.otherFunc(&modifiedURL)
	}
}

func (c *compiledConfig) SetEnv() {
	for name, val := range c.envVars {
		for _, actualName := range []string{strings.ToLower(name), strings.ToUpper(name)} {
			if err := os.Setenv(actualName, val); err != nil {
				log.Warnf("Error setting %s environment variable: %v", actualName, err)
			}
		}
	}
}

func getProxyURL(envSetting string, endpointCfg proxyEndpointConfig) (*url.URL, error) {
	errs := errorhelpers.NewErrorList("could not determine a valid proxy URL")
	if envSetting != "" {
		u, err := url.Parse(envSetting)
		if u != nil {
			return u, nil
		}
		errs.AddWrap(err, "parsing setting from environment variable")
	}
	u, err := endpointCfg.toURL()
	if err != nil {
		errs.AddWrap(err, "parsing setting from config file")
	}
	return u, nil
}

func (c *proxyConfig) Compile(envCfg environmentConfig) *compiledConfig {
	allProxyURL, err := getProxyURL(envCfg.AllProxy, c.proxyEndpointConfig)
	if err != nil {
		log.Warnf("Failed to obtain default proxy configuration: %v", err)
	}
	httpProxyURL, err := getProxyURL(envCfg.HTTPProxy, c.HTTP)
	if err != nil {
		log.Warnf("Failed to obtain HTTP proxy configuration: %v", err)
	}
	if httpProxyURL == nil {
		httpProxyURL = allProxyURL
	}
	httpsProxyURL, err := getProxyURL(envCfg.HTTPSProxy, c.HTTP)
	if err != nil {
		log.Warnf("Failed to obtain HTTPS proxy configuration: %v", err)
	}
	if httpsProxyURL == nil {
		httpsProxyURL = httpProxyURL
	}

	// Set excludes (no_proxy)
	var allExcludes []string
	allExcludes = append(allExcludes, c.Excludes...)
	if !c.OmitDefaultExcludes {
		allExcludes = append(allExcludes, defaultExcludes...)
	}

	for _, elem := range strings.Split(envCfg.NoProxy, ",") {
		elem = strings.TrimSpace(elem)
		if elem == "" {
			continue
		}
		allExcludes = append(allExcludes, elem)
	}
	allExcludes = sliceutils.Unique(allExcludes)

	validExcludes := allExcludes[:0]
	for _, excl := range allExcludes {
		if excl == "" || strings.ContainsAny(excl, " \t\r\n") {
			log.Warnf("Invalid proxy exclusion %q, ignoring...", excl)
			continue
		}
		validExcludes = append(validExcludes, excl)
	}

	canonicalNoProxyStr := strings.Join(validExcludes, ",")

	baseCfg := httpproxy.Config{
		NoProxy: canonicalNoProxyStr,
	}

	httpCfg := baseCfg
	httpCfg.HTTPProxy = urlToStringOrEmpty(httpProxyURL)
	httpsCfg := baseCfg
	httpsCfg.HTTPSProxy = urlToStringOrEmpty(httpsProxyURL)
	otherCfg := baseCfg
	otherCfg.HTTPProxy = urlToStringOrEmpty(allProxyURL)

	cc := &compiledConfig{
		httpFunc:  httpCfg.ProxyFunc(),
		httpsFunc: httpsCfg.ProxyFunc(),
		otherFunc: otherCfg.ProxyFunc(),
		envVars: map[string]string{
			"http_proxy":  httpCfg.HTTPProxy,
			"https_proxy": httpsCfg.HTTPSProxy,
			"all_proxy":   otherCfg.HTTPProxy,
			"no_proxy":    baseCfg.NoProxy,
		},
	}

	return cc
}
