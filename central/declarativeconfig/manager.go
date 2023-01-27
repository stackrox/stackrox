package declarativeconfig

// Manager manages reconciling declarative configuration.
type Manager interface {
	WatchDeclarativeConfigDir()
}
