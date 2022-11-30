package phonehome

import (
	"context"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/userpass"
	erroxGRPC "github.com/stackrox/rox/pkg/errox/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	grpcError "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/set"
	pkgPH "github.com/stackrox/rox/pkg/telemetry/phonehome"
	"google.golang.org/grpc"
)

var (
	ignoredPaths = []string{"/v1/ping", "/v1/metadata", "/static/"}
)

func track(ctx context.Context, t pkgPH.Telemeter, err error, info *grpc.UnaryServerInfo, trackedPaths set.FrozenSet[string]) {
	userAgent, userID, path, code := getRequestDetails(ctx, t.GetID(), err, info)

	// Track the API path and error code of some requests:

	for _, ip := range ignoredPaths {
		if strings.HasPrefix(path, ip) {
			return
		}
	}

	if trackedPaths.Contains("*") || trackedPaths.Contains(path) {
		t.Track("API Call", userID, map[string]any{
			"Path":       path,
			"Code":       code,
			"User-Agent": userAgent,
		})
	}
}

func getRequestDetails(ctx context.Context, centralID string, err error, info *grpc.UnaryServerInfo) (userAgent string, userID string, method string, code int) {
	ri := requestinfo.FromContext(ctx)
	userAgent = strings.Join(ri.Metadata.Get("User-Agent"), ", ")

	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil {
		log.Debug("Cannot identify user from context: ", iderr)
	} else if userpass.IsLocalAdmin(id) {
		userID = "local:" + centralID + ":admin"
	} else {
		userID = pkgPH.HashUserID(id.UID())
	}

	if ri.HTTPRequest != nil && ri.HTTPRequest.URL != nil {
		method = ri.HTTPRequest.URL.Path
		code = grpcError.ErrToHTTPStatus(err)
	} else if info != nil {
		method = info.FullMethod
		code = int(erroxGRPC.RoxErrorToGRPCCode(err))
	} else {
		// Something not expected:
		method = "unknown"
		code = -1
	}
	return
}

// GetGRPCInterceptor returns an API interceptor function for GRPC requests.
func GetGRPCInterceptor(t pkgPH.Telemeter) grpc.UnaryServerInterceptor {
	trackedPaths := pkgPH.InstanceConfig().APIPaths

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		go track(ctx, t, err, info, trackedPaths)
		return resp, err
	}
}

func statusCodeToError(code *int) error {
	if code == nil || *code == http.StatusOK {
		return nil
	}
	return errors.Errorf("%d %s", *code, http.StatusText(*code))
}

// GetHTTPInterceptor returns an API interceptor function for HTTP requests.
func GetHTTPInterceptor(t pkgPH.Telemeter) httputil.HTTPInterceptor {
	trackedPaths := pkgPH.InstanceConfig().APIPaths

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			statusTrackingWriter := httputil.NewStatusTrackingWriter(w)
			handler.ServeHTTP(statusTrackingWriter, r)
			go track(r.Context(), t, statusCodeToError(statusTrackingWriter.GetStatusCode()), nil, trackedPaths)
		})
	}
}
