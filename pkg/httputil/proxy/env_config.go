package proxy

import (
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/http/httpproxy"
)

type environmentConfig struct {
	httpproxy.Config
	AllProxy string
}

func readConfigFromEnv() environmentConfig {
	allProxy := ""
	for _, varName := range []string{"ALL_PROXY", "all_proxy"} {
		if val := os.Getenv(varName); val != "" {
			allProxy = val
			break
		}
	}
	cfg := environmentConfig{
		Config:   *httpproxy.FromEnvironment(),
		AllProxy: allProxy,
	}

	vars := map[string]*string{
		"http_proxy":  &cfg.HTTPProxy,
		"https_proxy": &cfg.HTTPSProxy,
		"all_proxy":   &cfg.AllProxy,
	}

	for name, val := range vars {
		if *val == "" {
			continue
		}
		if _, err := url.Parse(*val); err != nil {
			log.Warnf("Invalid setting %q for %s/%s environment variable: %v. Ignoring setting", *val, name, strings.ToUpper(name), err)
			*val = ""
		}
	}

	return cfg
}

var (
	initialEnvCfg = readConfigFromEnv()
)
