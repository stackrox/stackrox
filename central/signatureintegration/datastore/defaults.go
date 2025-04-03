package datastore

import (
	rekorClient "github.com/sigstore/rekor/pkg/generated/client"
	"github.com/stackrox/rox/generated/storage"
)

func applyDefaultValues(integration *storage.SignatureIntegration) {
	if tlog := integration.GetTransparencyLog(); tlog.GetEnabled() {
		if tlog.GetUrl() == "" {
			tlog.Url = rekorClient.DefaultHost
		}
	}
}
