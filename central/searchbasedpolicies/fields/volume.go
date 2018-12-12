package fields

import (
	"strconv"

	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var (
	volumeQueryBuilder = builders.RegexQueryBuilder{
		RegexFields: []builders.RegexField{
			{
				FieldLabel:     search.VolumeName,
				FieldHumanName: "Volume name",
				RetrieveFieldValue: func(fields *storage.PolicyFields) string {
					return fields.GetVolumePolicy().GetName()
				},
			},
			{
				FieldLabel:     search.VolumeSource,
				FieldHumanName: "Volume source",
				RetrieveFieldValue: func(fields *storage.PolicyFields) string {
					return fields.GetVolumePolicy().GetSource()
				},
			},
			{
				FieldLabel:     search.VolumeDestination,
				FieldHumanName: "Volume destination",
				RetrieveFieldValue: func(fields *storage.PolicyFields) string {
					return fields.GetVolumePolicy().GetDestination()
				},
			},
			{
				FieldLabel:     search.VolumeReadonly,
				FieldHumanName: "Volume read-only",
				RetrieveFieldValue: func(fields *storage.PolicyFields) string {
					if fields.GetVolumePolicy().GetSetReadOnly() == nil {
						return ""
					}
					return strconv.FormatBool(fields.GetVolumePolicy().GetReadOnly())
				},
			},
			{
				FieldLabel:     search.VolumeType,
				FieldHumanName: "Volume type",
				RetrieveFieldValue: func(fields *storage.PolicyFields) string {
					return fields.GetVolumePolicy().GetType()
				},
			},
		},
	}
)
