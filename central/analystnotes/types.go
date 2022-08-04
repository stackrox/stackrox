package analystnotes

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	processNoteKeyNS = uuid.FromStringOrPanic("c19c0cea-b5df-40c4-80e7-836a1b0785e6")
)

// ProcessNoteKey is the key used for process notes.
// Different processes with the same key will have an identical
// set of comments/tags stored.
type ProcessNoteKey struct {
	DeploymentID  string
	ContainerName string
	ExecFilePath  string
	Args          string
}

// Serialize serializes the given process key (minus the deploymentID).
// CAREFUL: It does NOT check that the key is well-formed.
// Clients must call Validate separately.
func (k *ProcessNoteKey) Serialize() []byte {
	return []byte(uuid.NewV5(processNoteKeyNS, fmt.Sprintf("%s\x00%s\x00:%s", k.ContainerName, k.ExecFilePath, k.Args)).String())
}

// Validate validates that a ProcessNoteKey is well-formed.
func (k *ProcessNoteKey) Validate() error {
	if k == nil {
		return errors.New("process comment key is nil")
	}
	// It's okay for k.Args to be empty.
	if stringutils.AtLeastOneEmpty(k.DeploymentID, k.ContainerName, k.ExecFilePath) {
		return errors.Errorf("invalid process key %v: has missing fields", k)
	}
	return nil
}
