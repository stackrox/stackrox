package fixtures

import (
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// Test fixtures for tests involving whitelists

// GetProcessWhitelist returns an empty process whitelist with a random container name and deployment ID
func GetProcessWhitelist() *storage.ProcessWhitelist {
	createStamp, _ := ptypes.TimestampProto(time.Now())
	processName := uuid.NewV4().String()
	process := &storage.WhitelistElement{
		Element: &storage.WhitelistElement_ProcessName{
			ProcessName: processName,
		},
		Auto: true,
	}
	return &storage.ProcessWhitelist{
		Elements: []*storage.WhitelistElement{process},
		Created:  createStamp,
	}
}

// GetProcessWhitelistWithID returns a whitelist with the ID filled out
func GetProcessWhitelistWithID() *storage.ProcessWhitelist {
	whitelist := GetProcessWhitelistWithKey()
	whitelist.Id = uuid.NewV4().String()
	return whitelist
}

// GetProcessWhitelistWithKey returns a whitelist and its key.
func GetProcessWhitelistWithKey() *storage.ProcessWhitelist {
	key := &storage.ProcessWhitelistKey{
		DeploymentId:  uuid.NewV4().String(),
		ContainerName: uuid.NewV4().String(),
	}
	whitelist := GetProcessWhitelist()
	whitelist.Key = key
	return whitelist
}

// GetWhitelistElement returns a *storage.WhitelistElement with a given process name
func GetWhitelistElement(processName string) *storage.WhitelistElement {
	return &storage.WhitelistElement{
		Element: &storage.WhitelistElement_ProcessName{
			ProcessName: processName,
		},
		Auto: true,
	}
}
