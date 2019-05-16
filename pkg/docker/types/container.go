package types

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
)

// ContainerList is a trimmed down version of Docker types.Container
// easyjson:json
type ContainerList struct {
	ID     string `json:"Id"`
	Labels map[string]string
}

// ContainerJSON is a trimmed down version of Docker ContainerJSON
// easyjson:json
type ContainerJSON struct {
	*ContainerJSONBase `json:",omitempty"`
	Mounts             []MountPoint     `json:",omitempty"`
	Config             *Config          `json:",omitempty"`
	NetworkSettings    *NetworkSettings `json:",omitempty"`
}

// ContainerJSONBase is a trimmed down version of Docker ContainerJSONBase
type ContainerJSONBase struct {
	ID              string          `json:"Id"`
	Image           string          `json:",omitempty"`
	State           *ContainerState `json:",omitempty"`
	Name            string          `json:",omitempty"`
	AppArmorProfile string          `json:",omitempty"`
	HostConfig      *HostConfig     `json:",omitempty"`
}

// HostConfig is a trimmed down version of Docker HostConfig
type HostConfig struct {
	CapAdd      strslice.StrSlice `json:",omitempty"` // List of kernel capabilities to add to the container
	CapDrop     strslice.StrSlice `json:",omitempty"` // List of kernel capabilities to remove from the container
	SecurityOpt []string          `json:",omitempty"` // List of string values to customize labels for MLS systems, such as SELinux.

	Resources

	NetworkMode    container.NetworkMode   `json:",omitempty"` // Network mode to use for the container
	RestartPolicy  container.RestartPolicy `json:",omitempty"` // Restart policy to be used for the container
	IpcMode        container.IpcMode       `json:",omitempty"` // IPC namespace to use for the container
	PidMode        container.PidMode       `json:",omitempty"` // PID namespace to use for the container
	Privileged     bool                    `json:",omitempty"` // Is the container in privileged mode
	ReadonlyRootfs bool                    `json:",omitempty"` // Is the container root filesystem in read-only
	UTSMode        container.UTSMode       `json:",omitempty"` // UTS namespace to use for the container
	UsernsMode     container.UsernsMode    `json:",omitempty"` // The user namespace to use for the container
}

// Resources is a trimmed down version of Docker Resources
type Resources struct {
	CgroupParent string `json:",omitempty"` // Parent cgroup.

	// Applicable to all platforms
	CPUShares int64 `json:"CpuShares"`  // CPU shares (relative weight vs. other containers)
	Memory    int64 `json:",omitempty"` // Memory limit (in bytes)

	Devices   []container.DeviceMapping `json:",omitempty"` // List of devices to map inside the container
	PidsLimit int64                     `json:",omitempty"` // Setting pids limit for a container
	Ulimits   []*units.Ulimit           `json:",omitempty"` // List of ulimits to be set in the container
}

// ContainerState is a trimmed down version of Docker ContainerState
type ContainerState struct {
	Running bool    `json:",omitempty"`
	Health  *Health `json:",omitempty"`
}

// MountPoint is a trimmed down version of Docker MountPoint
type MountPoint struct {
	Type        mount.Type        `json:",omitempty"`
	Name        string            `json:",omitempty"`
	Source      string            `json:",omitempty"`
	Destination string            `json:",omitempty"`
	Driver      string            `json:",omitempty"`
	Mode        string            `json:",omitempty"`
	Propagation mount.Propagation `json:",omitempty"`
}

// NetworkSettings is a trimmed down version of Docker NetworkSettings
type NetworkSettings struct {
	NetworkSettingsBase `json:",omitempty"`
	Networks            map[string]struct{} `json:",omitempty"`
}

// NetworkSettingsBase is a trimmed down version of Docker NetworkSettingsBase
type NetworkSettingsBase struct {
	Ports nat.PortMap `json:",omitempty"` // Ports is a collection of PortBinding indexed by Port
}

// Health is a trimmed down version of Docker Health
type Health struct {
	Status string `json:",omitempty"` // Status is one of Starting, Healthy or Unhealthy
}
