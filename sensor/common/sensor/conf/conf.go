package conf

import (
	"fmt"
	"net"
	"strings"

	"github.com/stackrox/rox/pkg/env"
)

var (
	CentralEndpoint string
)

const (
	defaultHttpsPort = 443
	defaultHttpPort  = 80
	missingPortErr   = "missing port in address"
)

func ensureHasPort(addr string, port int) (string, error) {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		if !strings.Contains(err.Error(), missingPortErr) {
			return "", err
		}

		addr = fmt.Sprintf("%s:%d", addr, port)
		if _, _, err := net.SplitHostPort(addr); err != nil {
			// Still broken after adding a port, something else must be off as well.
			return "", err
		}
	}

	return addr, nil
}

func initCentralEndpoint(endpoint string) {
	CentralEndpoint = endpoint
	if strings.HasPrefix(CentralEndpoint, "https://") {
		CentralEndpoint = strings.TrimPrefix(CentralEndpoint, "https://")
		if addr, err := ensureHasPort(CentralEndpoint, defaultHttpsPort); err == nil {
			CentralEndpoint = addr
		}
	} else if strings.HasPrefix(CentralEndpoint, "http://") {
		CentralEndpoint = strings.TrimPrefix(CentralEndpoint, "http://")
		if addr, err := ensureHasPort(CentralEndpoint, defaultHttpPort); err == nil {
			CentralEndpoint = addr
		}
	} else {
		if addr, err := ensureHasPort(CentralEndpoint, defaultHttpsPort); err == nil {
			CentralEndpoint = addr
		}
	}
}

func init() {
	initCentralEndpoint(env.CentralEndpoint.Setting())
}

func Reinit() {
	initCentralEndpoint(env.CentralEndpoint.Setting())
}
