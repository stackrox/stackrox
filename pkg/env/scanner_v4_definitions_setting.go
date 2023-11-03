package env

import "time"

var (
	// RepoMappingUpdateMaxInitialWait is the maximum wait time before the first attempt to download data mapping cpes to repos.
	RepoMappingUpdateMaxInitialWait = registerDurationSetting("ROX_MAPPING_UPDATE_MAX_INITIAL_WAIT", 3*time.Minute)
)
