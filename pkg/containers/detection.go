package containers

import "os"

// IsRunningInContainer checks whether the current process is being executed inside a docker container
func IsRunningInContainer() bool {
	return isDocker() || isPodman() || isCryoSpark()
}

func isDocker() bool {
	return pathExists("/.dockerenv") ||
		pathExists("/.dockerinit") ||
		pathExists("/run/.containerenv") ||
		pathExists("/var/run/.containerenv")
}

func isPodman() bool {
	return os.Getenv("container") == "oci" || os.Getenv("container") == "podman"
}

func isCryoSpark() bool {
	_, hostnameSet := os.LookupEnv("CRYOSPARC_MASTER_HOSTNAME")
	_, userSet := os.LookupEnv("CRYOSPARC_USER")
	return hostnameSet || userSet || pathExists("/opt/cryosparc") || pathExists("/app/cryosparc")
}

func pathExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}
