package fields

import (
	"strconv"

	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

var (
	portQueryBuilder = builders.RegexQueryBuilder{
		RegexFields: []builders.RegexField{
			{
				FieldLabel:     search.Port,
				FieldHumanName: "Port",
				RetrieveFieldValue: func(fields *v1.PolicyFields) string {
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
				RetrieveFieldValue: func(fields *v1.PolicyFields) string {
					return fields.GetPortPolicy().GetProtocol()
				},
			},
		},
	}
)
