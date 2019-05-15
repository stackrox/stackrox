package fields

import (
	"strconv"

	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// PortQueryBuilder is a regex query builder for the ports used in a deployment.
	PortQueryBuilder = builders.RegexQueryBuilder{
		RegexFields: []builders.RegexField{
			{
				FieldLabel:     search.Port,
				FieldHumanName: "Port",
				RetrieveFieldValue: func(fields *storage.PolicyFields) string {
					port := fields.GetPortPolicy().GetPort()
					if port == 0 {
						return ""
					}
					return strconv.FormatInt(int64(port), 10)
				},
			},
			{
				FieldLabel:     search.PortProtocol,
				FieldHumanName: "Protocol",
				RetrieveFieldValue: func(fields *storage.PolicyFields) string {
					return fields.GetPortPolicy().GetProtocol()
				},
			},
		},
	}
)
