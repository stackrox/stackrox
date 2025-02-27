package continuousprofiling

import "github.com/grafana/pyroscope-go"

var (
	acsToPyroscopeProfileType = map[ProfileType]pyroscope.ProfileType{
		ProfileCPU:           pyroscope.ProfileCPU,
		ProfileAllocObjects:  pyroscope.ProfileAllocObjects,
		ProfileAllocSpace:    pyroscope.ProfileAllocSpace,
		ProfileInuseObjects:  pyroscope.ProfileInuseObjects,
		ProfileInuseSpace:    pyroscope.ProfileInuseSpace,
		ProfileGoroutines:    pyroscope.ProfileGoroutines,
		ProfileMutexCount:    pyroscope.ProfileMutexCount,
		ProfileMutexDuration: pyroscope.ProfileMutexDuration,
		ProfileBlockCount:    pyroscope.ProfileBlockCount,
		ProfileBlockDuration: pyroscope.ProfileBlockDuration,
	}
)

func convertToPyroscopeProfileTypes(profileTypes []ProfileType) ([]pyroscope.ProfileType, error) {
	pyroscopeProfileTypes := make([]pyroscope.ProfileType, 0, len(profileTypes))

	if len(profileTypes) == 0 {
		return pyroscopeProfileTypes, ErrAtLeastOneProfileIsNeeded
	}

	for _, profile := range profileTypes {
		pyroscopeProfile, found := acsToPyroscopeProfileType[profile]
		if !found {
			return pyroscopeProfileTypes, ErrUnknownProfileType
		}
		pyroscopeProfileTypes = append(pyroscopeProfileTypes, pyroscopeProfile)
	}
	return pyroscopeProfileTypes, nil
}

func convertToPyroscopeConfig(cfg *ProfilerConfiguration) (*pyroscope.Config, error) {
	if cfg.ApplicationName == "" {
		return nil, ErrApplicationName
	}
	if cfg.ServerAddress == "" {
		return nil, ErrServerAddress
	}
	pyroscopeProfileTypes, err := convertToPyroscopeProfileTypes(cfg.ProfilerTypes)
	if err != nil {
		return nil, err
	}
	pyroscopeConfig := &pyroscope.Config{
		ApplicationName:   cfg.ApplicationName,
		ServerAddress:     cfg.ServerAddress,
		BasicAuthUser:     cfg.BasicAuthUser,
		BasicAuthPassword: cfg.BasicAuthPassword,
		ProfileTypes:      pyroscopeProfileTypes,
	}
	if cfg.WithLogs {
		pyroscopeConfig.Logger = log
	}
	return pyroscopeConfig, nil
}
