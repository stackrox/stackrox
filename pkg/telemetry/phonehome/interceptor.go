package phonehome

import (
	"context"
	"net/http"
	"strings"

	erroxGRPC "github.com/stackrox/rox/pkg/errox/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	grpcError "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

const userAgentKey = "User-Agent"

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

	// This is either the gRPC client or the grpc-gateway user agent:
	grpcClientAgent := ri.Metadata.Get(userAgentKey)

	// Use the wrapped HTTP request if provided by the grpc-gateway.
	if ri.HTTPRequest != nil {
		var path string
		if ri.HTTPRequest.URL != nil {
			path = ri.HTTPRequest.URL.Path
		}
		if clientAgent := ri.HTTPRequest.Headers.Get(userAgentKey); clientAgent != "" {
			grpcClientAgent = append(grpcClientAgent, clientAgent)
		}
		return &RequestParams{
			UserAgent: strings.Join(grpcClientAgent, " "),
			UserID:    id,
			Method:    ri.HTTPRequest.Method,
			Path:      path,
			Code:      grpcError.ErrToHTTPStatus(err),
			GRPCReq:   req,
			Headers:   headers(ri.HTTPRequest.Headers),
		}
	}

	return &RequestParams{
		UserAgent: strings.Join(grpcClientAgent, " "),
		UserID:    id,
		Method:    grpcFullMethod,
		Path:      grpcFullMethod,
		Code:      int(erroxGRPC.RoxErrorToGRPCCode(err)),
		GRPCReq:   req,
		Headers:   ri.Metadata,
	}
}

func getHTTPRequestDetails(ctx context.Context, r *http.Request, status int) *RequestParams {
	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil {
		log.Debug("Cannot identify user from context: ", iderr)
	}

	return &RequestParams{
		UserAgent: r.Header.Get(userAgentKey),
		UserID:    id,
		Method:    r.Method,
		Path:      r.URL.Path,
		Code:      status,
		HTTPReq:   r,
		Headers:   headers(r.Header),
	}
}
