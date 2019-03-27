package volumes

import (
	v1 "k8s.io/api/core/v1"
)

const gitRepoType = "GitRepo"

type gitRepo struct {
	*v1.GitRepoVolumeSource
}

func (h *gitRepo) Source() string {
	return h.Repository
}

func (h *gitRepo) Type() string {
	return gitRepoType
}

func createGitRepo(i interface{}) VolumeSource {
	gitVolume, ok := i.(*v1.GitRepoVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &gitRepo{
		GitRepoVolumeSource: gitVolume,
	}
}

func init() {
	VolumeRegistry[gitRepoType] = createGitRepo
}
