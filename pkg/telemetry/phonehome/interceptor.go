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

func getGRPCRequestDetails(ctx context.Context, err error, info *grpc.UnaryServerInfo, req any) *RequestParams {
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
		GRPCReq:   req,
	}
}

func getHTTPRequestDetails(ctx context.Context, r *http.Request, err error) *RequestParams {
	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil {
		log.Debug("Cannot identify user from context: ", iderr)
	}

	return &RequestParams{
		UserAgent: strings.Join(r.Header.Values("User-Agent"), ", "),
		UserID:    id,
		Path:      r.URL.Path,
		Code:      grpcError.ErrToHTTPStatus(err),
		HTTPReq:   r,
	}
}
