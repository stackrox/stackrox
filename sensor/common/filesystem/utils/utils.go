package utils

import "github.com/stackrox/rox/generated/storage"

func IsNodeFileAccess(fileAccess *storage.FileAccess) bool {
	return fileAccess.GetProcess().GetDeploymentId() == ""
}

func IsDeploymentFileAccess(fileAccess *storage.FileAccess) bool {
	return !IsNodeFileAccess(fileAccess)
}
