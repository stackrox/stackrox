package rest

import (
	"encoding/json"
	"net/http"
	"strings"

	imageCVEDS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	"github.com/stackrox/rox/central/views/imagecveflat"
	vulnReqDataStore "github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

type handler struct {
	cveView      imagecveflat.CveFlatView
	cveDS        imageCVEDS.DataStore
	vulnReqStore vulnReqDataStore.DataStore
}

// NewHandler creates a new REST handler for vulnerability management endpoints.
func NewHandler(cveView imagecveflat.CveFlatView, cveDS imageCVEDS.DataStore, vulnReqStore vulnReqDataStore.DataStore) http.Handler {
	return &handler{
		cveView:      cveView,
		cveDS:        cveDS,
		vulnReqStore: vulnReqStore,
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !features.VulnMgmtRESTAPI.Enabled() {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Route: /api/v2/vulnmgmt/cves or /api/v2/vulnmgmt/cves/{cve}/detail
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/vulnmgmt/cves")
	path = strings.TrimPrefix(path, "/")

	switch {
	case path == "" || path == "/":
		h.listCVEs(w, r)
	case strings.HasSuffix(path, "/detail"):
		cve := strings.TrimSuffix(path, "/detail")
		if cve == "" {
			http.Error(w, "CVE identifier required", http.StatusBadRequest)
			return
		}
		h.getCVEDetail(w, r, cve)
	default:
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Errorf("failed to encode JSON response: %v", err)
	}
}
