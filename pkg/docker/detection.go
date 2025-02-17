package docker

import "os"

// IsRunningInDocker checks whether the current process is being executed inside a docker container
func IsRunningInDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}
