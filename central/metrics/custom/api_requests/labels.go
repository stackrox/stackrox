package api_requests

import (
	"strconv"
	"strings"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/pkg/clientprofile"
	"github.com/stackrox/rox/pkg/glob"
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

// headerPatternToLabelPattern converts an HTTP header name pattern to a
// Prometheus label pattern by removing hyphens.
// E.g. "Rh-Servicenow-*" -> "RhServicenow*".
func labelMatchesHeaderPattern(label tracker.Label, header glob.Pattern) bool {
	p := glob.Pattern(strings.ReplaceAll(string(header), "-", ""))
	return p.Match(string(label))
}

// makeHeaderGetter creates a getter that returns filtered values of the request
// header matching the given label. headerPattern and valuePattern are the
// profile header entry that matched the label at configuration time.
func makeHeaderGetter(headerPattern, valuePattern glob.Pattern, label tracker.Label) tracker.Getter[*finding] {
	return func(f *finding) string {
		for h, values := range clientprofile.Headers(f.Headers).GetMatching(headerPattern, valuePattern) {
			if compareIgnoringDash(h, label) {
				return strings.Join(values, "; ")
			}
		}
		return ""
	}
}

// compareIgnoringDash reports whether header equals label when dashes in
// header are skipped.
func compareIgnoringDash(header string, label tracker.Label) bool {
	li := 0
	for i := range len(header) {
		if header[i] == '-' {
			continue
		}
		if li >= len(label) || header[i] != label[li] {
			return false
		}
		li++
	}
	return li == len(label)
}
