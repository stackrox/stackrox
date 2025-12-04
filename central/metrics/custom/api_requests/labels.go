package api_requests

import (
	"strconv"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var LazyLabels = tracker.LazyLabelGetters[*finding]{
	"UserID": func(f *finding) string {
		if f.UserID != nil {
			return f.UserID.UID()
		}
		return ""
	},
	"UserAgent": func(f *finding) string { return getUserAgentFromHeaders(f.Headers) },
	"Path":      func(f *finding) string { return f.Path },
	"Method":    func(f *finding) string { return f.Method },
	"Status":    func(f *finding) string { return strconv.Itoa(f.Code) },
}

type finding = phonehome.RequestParams

func getUserAgentFromHeaders(headers func(string) []string) string {
	if headers == nil {
		return ""
	}
	if userAgents := headers("User-Agent"); len(userAgents) > 0 {
		return userAgents[0]
	}
	if userAgents := headers("user-agent"); len(userAgents) > 0 {
		return userAgents[0]
	}
	return ""
}
