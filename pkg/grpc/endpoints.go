package grpc

import (
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// EndpointsConfig configures the endpoints for exposure of the gRPC/HTTP services.
type EndpointsConfig struct {
	// MultiplexedEndpoints are the endpoints on which to serve a multiplexed gRPC/HTTP service.
	MultiplexedEndpoints []string
	// HTTPEndpoints are the endpoints on which to serve HTTP only.
	HTTPEndpoints []string
	// GRPCEndpoints are the endpoints on which to serve GRPC only.
	GRPCEndpoints []string
}

// checkPortNames checks that all ports referenced in the given endpoints of the specified kind are known.
func checkPortNames(endpoints []string, kind string) []error {
	var errs []error
	for _, ep := range endpoints {
		_, port, err := net.SplitHostPort(ep)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "could not parse %s endpoint %q", kind, ep))
			continue
		}
		_, err = net.LookupPort("tcp", port)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "invalid port %q in %s endpoint %q", port, kind, ep))
			continue
		}
	}
	return errs
}

// Validate checks that all port names are known.
func (p *EndpointsConfig) Validate() error {
	errList := errorhelpers.NewErrorList("port setting validation")
	errList.AddErrors(checkPortNames(p.MultiplexedEndpoints, "multiplexed")...)
	errList.AddErrors(checkPortNames(p.HTTPEndpoints, "HTTP")...)
	errList.AddErrors(checkPortNames(p.GRPCEndpoints, "gRPC")...)
	return errList.ToError()
}

// asEndpoint returns an all-interface endpoint of form `:<port>` if `portOrEndpoint` is a port only (does not contain
// a ':'). Otherwise, `portOrEndpoint` is returned as-is.
func asEndpoint(portOrEndpoint string) string {
	if !strings.ContainsRune(portOrEndpoint, ':') {
		return fmt.Sprintf(":%s", portOrEndpoint)
	}
	return portOrEndpoint
}

// AddFromParsedSpec parses a specification string, i.e., a comma-separated list of `[type@]endpoint` strings (where
// type is `grpc` or `http`, and endpoint can be `<port>` or `[address]:<port>`).
func (p *EndpointsConfig) AddFromParsedSpec(fullSpecStr string) error {
	specs := strings.Split(fullSpecStr, ",")
	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		parts := strings.SplitN(spec, "@", 2)
		if len(parts) == 1 {
			p.MultiplexedEndpoints = append(p.MultiplexedEndpoints, asEndpoint(parts[0]))
			continue
		}
		switch typ := strings.TrimSpace(strings.ToLower(parts[0])); typ {
		case "http":
			p.HTTPEndpoints = append(p.HTTPEndpoints, asEndpoint(strings.TrimSpace(parts[1])))
		case "grpc":
			p.GRPCEndpoints = append(p.GRPCEndpoints, asEndpoint(strings.TrimSpace(parts[1])))
		default:
			return errors.Errorf("unknown endpoint type %q", typ)
		}
	}
	return nil
}
