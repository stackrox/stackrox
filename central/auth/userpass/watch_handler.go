package userpass

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/htpasswd"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/utils"
)

type watchHandler struct {
	manager *basic.Manager
}

func (h *watchHandler) OnChange(dir string) (interface{}, error) {
	htpasswdPath := filepath.Join(dir, htpasswdFile)
	f, err := os.Open(htpasswdPath)
	if err != nil {
		return nil, errors.Wrap(err, "opening htpasswd file")
	}
	defer utils.IgnoreError(f.Close)

	return htpasswd.ReadHashFile(f)
}

func (h *watchHandler) OnStableUpdate(val interface{}, err error) {
	var hashFile *htpasswd.HashFile
	if err != nil {
		log.Warnf("Error reading htpasswd file: %v. Basic (username/password) auth will be disabled until the issue is remediated", err)
		log.Warn("Note: if you intended to disable basic auth, you can suppress this warning by populating the secret with an empty htpasswd file")
	} else {
		hashFile, _ = val.(*htpasswd.HashFile)
		if hashFile == nil {
			log.Info("No htpasswd file found. Disabling basic auth")
		} else {
			log.Info("htpasswd file found. Updating basic auth credentials")
		}
	}
	h.manager.SetHashFile(hashFile)
}

func (h *watchHandler) OnWatchError(err error) {
	log.Errorf("Error watching for htpasswd file changes: %v", err)
}
