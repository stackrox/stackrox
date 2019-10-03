package k8scfgwatch

// Handler handles events related to watched Kubernetes config (ConfigMap or Secret) mount directories.
type Handler interface {
	// OnChange is invoked whenever a change of the configuration data is detected. The argument directory is a
	// directory containing a stable view of the data, except for deletion of the entire directory.
	// Implementors should ensure this method is side-effect free; any side effects of reading new values should be
	// deferred to `OnStableUpdate`.
	OnChange(dir string) (interface{}, error)

	// OnStableUpdate is invoked with the return values of `OnChange` whenever the input directory passed to `OnChange`
	// has been deemed, stable, i.e., was still current after `OnChange` returned.
	OnStableUpdate(val interface{}, err error)
	// OnWatchError is invoked whenever there was an error watching the configuration directory. If, e.g., the directory
	// does not exist or is not a ConfigMap/Secret volume mount, this method is invoked at every poll interval. If you
	// do not want this, wrap the handler via a call to `DeduplicateWatchErrors`.
	OnWatchError(err error)
}
