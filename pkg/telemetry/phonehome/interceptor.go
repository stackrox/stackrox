package phonehome

import (
	"context"
	"net/http"
	"strings"
	"sync"

	erroxGRPC "github.com/stackrox/rox/pkg/errox/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	grpcError "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"google.golang.org/grpc"
)

// RequestParams holds intercepted call parameters.
type RequestParams struct {
	UserAgent string
	UserID    authn.Identity
	Path      string
	Code      int
	GrpcReq   any
	HttpReq   *http.Request
}

var (
	mux = &sync.Mutex{}
)

func (cfg *Config) track(rp *RequestParams) {
	id := cfg.HashUserAuthID(rp.UserID)
	for event, funcs := range cfg.interceptors {
		props := map[string]any{}
		ok := true
		for _, interceptor := range funcs {
			if ok = interceptor(rp, props); !ok {
				break
			}
		}
		if ok {
			cfg.telemeter.Track(event, id, props)
		}
	}
}

func getGrpcRequestDetails(ctx context.Context, err error, info *grpc.UnaryServerInfo, req any) *RequestParams {
	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil {
		log.Debug("Cannot identify user from context: ", iderr)
	}

	ri := requestinfo.FromContext(ctx)
	return &RequestParams{
		UserAgent: strings.Join(ri.Metadata.Get("User-Agent"), ", "),
		UserID:    id,
		Path:      info.FullMethod,
		Code:      int(erroxGRPC.RoxErrorToGRPCCode(err)),
		GrpcReq:   req,
	}
}

func getHttpRequestDetails(ctx context.Context, r *http.Request, err error) *RequestParams {
	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil {
		log.Debug("Cannot identify user from context: ", iderr)
	}

	return &RequestParams{
		UserAgent: strings.Join(r.Header.Values("User-Agent"), ", "),
		UserID:    id,
		Path:      r.URL.Path,
		Code:      grpcError.ErrToHTTPStatus(err),
		HttpReq:   r,
	}
}
