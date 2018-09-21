package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

var (
	commandQueryBuilder = builders.RegexQueryBuilder{
		RegexFields: []builders.RegexField{
			{
				FieldLabel:     search.Command,
				FieldHumanName: "Command",
				RetrieveFieldValue: func(fields *v1.PolicyFields) string {
					return fields.GetCommand()
				},
			},
		},
	}

	commandArgsQueryBuilder = builders.RegexQueryBuilder{
		RegexFields: []builders.RegexField{
			{
				FieldLabel:     search.CommandArgs,
				FieldHumanName: "Command args",
				RetrieveFieldValue: func(fields *v1.PolicyFields) string {
					return fields.GetArgs()
				},
			},
		},
	}

	directoryQueryBuilder = builders.RegexQueryBuilder{
		RegexFields: []builders.RegexField{
			{
				FieldLabel:     search.Directory,
				FieldHumanName: "Directory",
				RetrieveFieldValue: func(fields *v1.PolicyFields) string {
					return fields.GetDirectory()
				},
			},
		},
	}

	userQueryBuilder = builders.RegexQueryBuilder{
		RegexFields: []builders.RegexField{
			{
				FieldLabel:     search.User,
				FieldHumanName: "User",
				RetrieveFieldValue: func(fields *v1.PolicyFields) string {
					return fields.GetUser()
				},
			},
		},
	}
)
