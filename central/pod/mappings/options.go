package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap describes the options for Pods
var OptionsMap = search.Walk(v1.SearchCategory_PODS, "pod", (*storage.Pod)(nil))
