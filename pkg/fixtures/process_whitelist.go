package fixtures

import (
	"fmt"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// Test fixtures for tests involving whitelists

// GetProcessWhitelist returns an empty process whitelist with a random container name and deployment ID
func GetProcessWhitelist() *storage.ProcessWhitelist {
	createStamp, _ := ptypes.TimestampProto(time.Now())
	id := uuid.NewV4().String()
	processName := uuid.NewV4().String()
	process := &storage.Process{
		Name: processName,
		Auto: true,
	}
	return &storage.ProcessWhitelist{
		ContainerName: id[:16],
		DeploymentId:  id[16:],
		Processes:     []*storage.Process{process},
		Created:       createStamp,
	}
}

// GetProcessWhitelistWithID returns a whitelist with the ID filled out
func GetProcessWhitelistWithID() *storage.ProcessWhitelist {
	whitelist := GetProcessWhitelist()
	whitelist.Id = fmt.Sprintf("%s/%s", whitelist.DeploymentId, whitelist.ContainerName)
	return whitelist
}

// GetWhitelistProcess returns a *storage.Process with a given name
func GetWhitelistProcess(name string) *storage.Process {
	return &storage.Process{
		Name: name,
		Auto: true,
	}
}
