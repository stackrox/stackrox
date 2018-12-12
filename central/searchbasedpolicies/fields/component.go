package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var (
	componentQueryBuilder = builders.RegexQueryBuilder{
		RegexFields: []builders.RegexField{
			{
				FieldLabel:     search.Component,
				FieldHumanName: "Component name",
				RetrieveFieldValue: func(fields *storage.PolicyFields) string {
					return fields.GetComponent().GetName()
				},
			},
			{
				FieldLabel:     search.ComponentVersion,
				FieldHumanName: "Component version",
				RetrieveFieldValue: func(fields *storage.PolicyFields) string {
					return fields.GetComponent().GetVersion()
				},
			},
		},
	}
)
