package gcp

import "github.com/stackrox/rox/pkg/logging"

type gcpMetadata struct {
	ProjectID string
	Zone      string
}

var log = logging.LoggerForModule()
