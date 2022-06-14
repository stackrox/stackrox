package k8sutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	cgroupPrefixes = map[string]storage.ContainerRuntime{
		"docker": storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
		"crio":   storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME,
	}
)

// InferContainerRuntime tries to infer the container runtime of the running container by inspecting its own cgroups.
// A non-error return always corresponds to a known container runtime, while a return of UNKNOWN_CONTAINER_RUNTIME will
// always be accompanied by an error.
func InferContainerRuntime() (storage.ContainerRuntime, error) {
	cgroupFile := fmt.Sprintf("/proc/%d/cgroup", os.Getpid())
	f, err := os.Open(cgroupFile)
	if err != nil {
		return storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME, err
	}
	defer utils.IgnoreError(f.Close)

	return inferContainerRuntimeFromCGroupFile(f)
}

func inferContainerRuntimeFromCGroupFile(cgroupFile io.Reader) (storage.ContainerRuntime, error) {
	sc := bufio.NewScanner(cgroupFile)
	for sc.Scan() {
		line := sc.Text()
		parts := strings.SplitN(line, ":", 4)
		if len(parts) != 3 {
			continue
		}
		cgroupBaseName := path.Base(parts[2])
		cgroupBaseName = strings.TrimSuffix(cgroupBaseName, ".scope")

		cgroupParts := strings.SplitN(cgroupBaseName, "-", 2)
		if len(cgroupParts) != 2 {
			continue
		}
		if runtime, ok := cgroupPrefixes[cgroupParts[0]]; ok {
			return runtime, nil
		}
	}

	return storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME, errors.New("did not find a matching cgroup entry")
}
