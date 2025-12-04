package api_requests

import (
	"strconv"
	"strings"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/pkg/grpc/common/requestinterceptor"
)

type finding = requestinterceptor.RequestParams

// commonLabels are shared across all profile trackers.
var commonLabels = tracker.LazyLabelGetters[*finding]{
	"UserID": func(f *finding) string {
		if f.UserID != nil {
			return f.UserID.UID()
		}
		return ""
	},
	"Path":   func(f *finding) string { return f.Path },
	"Method": func(f *finding) string { return f.Method },
	"Status": func(f *finding) string { return strconv.Itoa(f.Code) },
}

// headerToLabel converts an HTTP header name to a valid Prometheus label name
// by removing hyphens. E.g. "Rh-Servicenow-Instance" -> "RhServicenowInstance".
func headerToLabel(header string) tracker.Label {
	return tracker.Label(strings.ReplaceAll(header, "-", ""))
}

// makeHeaderGetter creates a getter that finds the request header whose
// hyphen-stripped name matches the label and returns its value.
func makeHeaderGetter(label tracker.Label) tracker.Getter[*finding] {
	return func(f *finding) string {
		for h := range f.Headers {
			if headerToLabel(h) == label {
				return strings.Join(f.Headers.Values(h), "; ")
			}
		}
		return ""
	}
}
