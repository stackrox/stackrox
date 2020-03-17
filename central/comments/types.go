package comments

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

// ProcessCommentKey is the key used for process comments.
// Different processes with the same comment key will have an identical
// set of comments stored.
type ProcessCommentKey struct {
	DeploymentID  string
	ContainerName string
	ExecFilePath  string
	Args          string
}

// Validate validates that a ProcessCommentKey is well-formed.
func (k *ProcessCommentKey) Validate() error {
	if k == nil {
		return errors.New("process comment key is nil")
	}
	// It's okay for k.Args to be empty.
	if stringutils.AtLeastOneEmpty(k.DeploymentID, k.ContainerName, k.ExecFilePath) {
		return errors.Errorf("invalid process key %v: has missing fields", k)
	}
	return nil
}

// ProcessToKey converts a process indicator to the key used for comments.
func ProcessToKey(indicator *storage.ProcessIndicator) *ProcessCommentKey {
	return &ProcessCommentKey{
		DeploymentID:  indicator.GetDeploymentId(),
		ContainerName: indicator.GetContainerName(),
		ExecFilePath:  indicator.GetSignal().GetExecFilePath(),
		Args:          indicator.GetSignal().GetArgs(),
	}
}
