package handler

import (
	"net/http"

	"bitbucket.org/stack-rox/apollo/central/logimbue/store"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// New returns a new instance of an HTTP handler for supporting logimbue.
func New(storage store.Store) http.Handler {
	return &handlerImpl{
		storage:            storage,
		compressorProvider: getCompressor,
	}
}
