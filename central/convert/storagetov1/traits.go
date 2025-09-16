package storagetov1

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func Traits(traits *storage.Traits) *v1.Traits {
	return &v1.Traits{
		MutabilityMode: convertMutabilityModeEnum(traits.GetMutabilityMode()),
		Visibility:     convertVisibilityEnum(traits.GetVisibility()),
		Origin:         convertOriginEnum(traits.GetOrigin()),
	}
}

func convertMutabilityModeEnum(val storage.Traits_MutabilityMode) v1.Traits_MutabilityMode {
	return v1.Traits_MutabilityMode(val)
}

func convertVisibilityEnum(val storage.Traits_Visibility) v1.Traits_Visibility {
	return v1.Traits_Visibility(val)
}

func convertOriginEnum(val storage.Traits_Origin) v1.Traits_Origin {
	return v1.Traits_Origin(val)
}
