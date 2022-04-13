package sensor

import (
	"net/http"

	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/httputil"
)

type readinessHandler struct {
	centralReachable *concurrency.Flag
}

func (h *readinessHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		httputil.WriteErrorf(w, http.StatusMethodNotAllowed, "unsupported method %q, only GET requests are allowed", req.Method)
		return
	}

	// TODO: We should mark ourselves ready only when central is reachable. However, this should be done
	// at a later point to decouple the introduction of this handler from changing the readiness logic.
	_, _ = w.Write([]byte("{}"))
}
