package containers

import "os"

// IsRunningInContainer checks whether the current process is being executed inside one of the containers we check for
func IsRunningInContainer() bool {
	return IsDocker() || IsPodman() || IsCryoSpark() || IsKube()
}

// IsDocker returns true if we are running in a docker instance
func IsDocker() bool {
	return pathExists("/.dockerenv") ||
		pathExists("/.dockerinit") ||
		pathExists("/run/.containerenv") ||
		pathExists("/var/run/.containerenv")
}

// IsKube returns true if we are running in a container managed by kubernetes
func IsKube() bool {
	return pathExists("/run/secrets/kubernetes.io/serviceaccount/namespace")
}

// IsPodman returns true if we are running in a podman instance
func IsPodman() bool {
	return os.Getenv("container") == "oci" || os.Getenv("container") == "podman"
}

// IsCryoSpark returns true if we are running in a cryospark instance
func IsCryoSpark() bool {
	_, hostnameSet := os.LookupEnv("CRYOSPARC_MASTER_HOSTNAME")
	_, userSet := os.LookupEnv("CRYOSPARC_USER")
	return hostnameSet || userSet || pathExists("/opt/cryosparc") || pathExists("/app/cryosparc")
}

func pathExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}
