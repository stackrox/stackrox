package server

import (
	"fmt"
	"net/http"

	"github.com/stackrox/rox/pkg/grpcweb"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/stringutils"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

func handleGRPC(w http.ResponseWriter, req *http.Request, validPaths set.StringSet, grpcSrv *grpc.Server) {
	// Check for HTTP/2.
	if req.ProtoMajor != 2 {
		if !validPaths.Contains(req.URL.Path) {
			// Client-streaming only works with HTTP/2.
			http.Error(w, "Method cannot be downgraded", http.StatusInternalServerError)
			return
		}
		req.ProtoMajor, req.ProtoMinor, req.Proto = 2, 0, "HTTP/2.0"
	}

	if req.Header.Get("TE") == "trailers" {
		// Yay, client accepts trailers! Let the normal gRPC handler handle the request.
		grpcSrv.ServeHTTP(w, req)
		return
	}

	acceptGRPCWeb := sliceutils.StringFind(req.Header["Accept"], "application/grpc-web") != -1
	if !acceptGRPCWeb {
		// Client doesn't support trailers and doesn't accept a response downgraded to gRPC web.
		http.Error(w, "Client neither supports trailers nor gRPC web responses", http.StatusInternalServerError)
		return
	}

	// Tell the server we would accept trailers (the gRPC server currently (v1.21.0) doesn't check for this but it
	// really should, as the purpose of the TE header according to the gRPC spec is to detect incompatible proxies).
	req.Header.Set("TE", "trailers")

	// Downgrade response to gRPC web.
	transcodingWriter, finalize := grpcweb.NewResponseWriter(w)
	grpcSrv.ServeHTTP(transcodingWriter, req)
	if err := finalize(); err != nil {
		log.Errorf("Error sending trailers in downgraded gRPC web response: %v", err)
	}
}

// CreateDowngradingHandler takes a gRPC server and a plain HTTP handler, and returns an HTTP handler that has the
// capability of handling requests that may require downgrading the response to gRPC web.
func CreateDowngradingHandler(grpcSrv *grpc.Server, httpHandler http.Handler) http.Handler {
	// Only allow paths corresponding to gRPC methods that do not use client streaming.
	validPaths := set.NewStringSet()

	for svcName, svcInfo := range grpcSrv.GetServiceInfo() {
		for _, methodInfo := range svcInfo.Methods {
			if methodInfo.IsClientStream {
				// Filter out client-streaming methods.
				continue
			}

			fullMethodName := fmt.Sprintf("/%s/%s", svcName, methodInfo.Name)
			validPaths.Add(fullMethodName)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if contentType, _ := stringutils.Split2(req.Header.Get("Content-Type"), "+"); contentType != "application/grpc" {
			httpHandler.ServeHTTP(w, req)
			return
		}

		handleGRPC(w, req, validPaths, grpcSrv)
	})
}
