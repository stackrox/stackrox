package storagetov1

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func Traits(traits *storage.Traits) *v1.Traits {
	traits2 := &v1.Traits{}
	traits2.SetMutabilityMode(convertMutabilityModeEnum(traits.GetMutabilityMode()))
	traits2.SetVisibility(convertVisibilityEnum(traits.GetVisibility()))
	traits2.SetOrigin(convertOriginEnum(traits.GetOrigin()))
	return traits2
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
