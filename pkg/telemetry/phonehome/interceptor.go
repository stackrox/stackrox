package phonehome

import (
	"context"
	"net/http"

	erroxGRPC "github.com/stackrox/rox/pkg/errox/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	grpcError "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

const userAgentHeaderKey = "User-Agent"

func (cfg *Config) track(rp *RequestParams) {
	cfg.interceptorsLock.RLock()
	defer cfg.interceptorsLock.RUnlock()
	if len(cfg.interceptors) == 0 {
		return
	}
	opts := []telemeter.Option{
		telemeter.WithUserID(cfg.HashUserAuthID(rp.UserID)),
		telemeter.WithGroups(cfg.GroupType, cfg.GroupID)}
	for event, funcs := range cfg.interceptors {
		props := map[string]any{}
		ok := true
		for _, interceptor := range funcs {
			if ok = interceptor(rp, props); !ok {
				break
			}
		}
		if ok {
			cfg.telemeter.Track(event, props, opts...)
		}
	}
}

func getGRPCRequestDetails(ctx context.Context, err error, grpcFullMethod string, req any) *RequestParams {
	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil {
		log.Debug("Cannot identify user from context: ", iderr)
	}

	ri := requestinfo.FromContext(ctx)

	// Use the wrapped HTTP request if provided by the grpc-gateway.
	if ri.HTTPRequest != nil {
		var path string
		if ri.HTTPRequest.URL != nil {
			path = ri.HTTPRequest.URL.Path
		}
		// This is either the gRPC client or the grpc-gateway user agent:
		grpcClientAgent := ri.Metadata.Get(userAgentHeaderKey)
		if clientAgent := ri.HTTPRequest.Headers.Get(userAgentHeaderKey); clientAgent != "" {
			grpcClientAgent = append(grpcClientAgent, clientAgent)
		}
		return &RequestParams{
			UserID:  id,
			Method:  ri.HTTPRequest.Method,
			Path:    path,
			Code:    grpcError.ErrToHTTPStatus(err),
			GRPCReq: req,
			Headers: func(key string) []string {
				if http.CanonicalHeaderKey(key) == userAgentHeaderKey {
					return grpcClientAgent
				}
				return Headers(ri.HTTPRequest.Headers).Get(key)
			},
		}
	}

	return &RequestParams{
		UserID:  id,
		Method:  grpcFullMethod,
		Path:    grpcFullMethod,
		Code:    int(erroxGRPC.RoxErrorToGRPCCode(err)),
		GRPCReq: req,
		Headers: ri.Metadata.Get,
	}
}

func getHTTPRequestDetails(ctx context.Context, r *http.Request, status int) *RequestParams {
	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil {
		log.Debug("Cannot identify user from context: ", iderr)
	}

	return &RequestParams{
		UserID:  id,
		Method:  r.Method,
		Path:    r.URL.Path,
		Code:    status,
		HTTPReq: r,
		Headers: Headers(r.Header).Get,
	}
}
