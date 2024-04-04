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

func getUserAgent(h requestinfo.HeaderGetter) string {
	return requestinfo.GetFirst(h, "User-Agent")
}

func getGRPCRequestDetails(ctx context.Context, err error, grpcFullMethod string, req any) *RequestParams {
	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil {
		log.Debug("Cannot identify user from context: ", iderr)
	}

	// Use the wrapped HTTP request details if provided:
	ri := requestinfo.FromContext(ctx)
	if ri.HTTPRequest != nil && ri.HTTPRequest.URL != nil {
		return &RequestParams{
			UserAgent: getUserAgent(ri.Metadata),
			UserID:    id,
			Method:    ri.HTTPRequest.Method,
			Path:      ri.HTTPRequest.URL.Path,
			Code:      grpcError.ErrToHTTPStatus(err),
			GRPCReq:   req,
			Header:    ri.Metadata,
		}
	}

	return &RequestParams{
		UserAgent: getUserAgent(ri.Metadata),
		UserID:    id,
		Method:    grpcFullMethod,
		Path:      grpcFullMethod,
		Code:      int(erroxGRPC.RoxErrorToGRPCCode(err)),
		GRPCReq:   req,
		Header:    ri.Metadata,
	}
}

func getHTTPRequestDetails(ctx context.Context, r *http.Request, status int) *RequestParams {
	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil {
		log.Debug("Cannot identify user from context: ", iderr)
	}
	header := requestinfo.WithGet(r.Header)
	return &RequestParams{
		UserAgent: getUserAgent(header),
		UserID:    id,
		Method:    r.Method,
		Path:      r.URL.Path,
		Code:      status,
		HTTPReq:   r,
		Header:    header,
	}
}
