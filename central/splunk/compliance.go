package splunk

import (
	"net/http"

	"github.com/stackrox/rox/pkg/httputil"
)

// NewComplianceHandler serves the retired Splunk TA Compliance feed.
//
// The route is kept so that existing TA configurations fail closed. Removing it outright
// would return 404, which a modular input is liable to treat as a transient error, and an
// empty 200 would be indexed as "no compliance findings" -- both hide the removal.
func NewComplianceHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		httputil.WriteErrorf(w, http.StatusGone,
			"the Splunk TA Compliance feed has been removed: Compliance 1.0 is no longer part of RHACS. "+
				"Remove the Compliance modular input from your Splunk TA configuration.")
	}
}
