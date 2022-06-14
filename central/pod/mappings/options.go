package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap describes the options for Pods
var OptionsMap = search.Walk(v1.SearchCategory_PODS, "pod", (*storage.Pod)(nil))
