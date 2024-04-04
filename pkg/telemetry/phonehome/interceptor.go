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

	ri := requestinfo.FromContext(ctx)
	rp := &RequestParams{
		UserAgent: getUserAgent(ri.Metadata),
		UserID:    id,
		GRPCReq:   req,
		Header:    ri.Metadata,
	}
	// Use the wrapped HTTP request details if provided:
	if ri.HTTPRequest != nil {
		rp.Method = ri.HTTPRequest.Method
		if ri.HTTPRequest.URL != nil {
			rp.Path = ri.HTTPRequest.URL.Path
		}
		rp.Code = grpcError.ErrToHTTPStatus(err)
		rp.Header = requestinfo.WithGet(ri.HTTPRequest.Headers)
	} else {
		rp.Method = grpcFullMethod
		rp.Path = grpcFullMethod
		rp.Code = int(erroxGRPC.RoxErrorToGRPCCode(err))
	}
	return rp
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
