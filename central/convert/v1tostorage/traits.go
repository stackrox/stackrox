package v1tostorage

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func Traits(traits *v1.Traits) *storage.Traits {
	return &storage.Traits{
		MutabilityMode: convertMutabilityModeEnum(traits.GetMutabilityMode()),
		Visibility:     convertVisibilityEnum(traits.GetVisibility()),
		Origin:         convertOriginEnum(traits.GetOrigin()),
	}
}

func convertMutabilityModeEnum(val v1.Traits_MutabilityMode) storage.Traits_MutabilityMode {
	return storage.Traits_MutabilityMode(val)
}

func convertVisibilityEnum(val v1.Traits_Visibility) storage.Traits_Visibility {
	return storage.Traits_Visibility(val)
}

func convertOriginEnum(val v1.Traits_Origin) storage.Traits_Origin {
	return storage.Traits_Origin(val)
}
