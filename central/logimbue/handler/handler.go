package handler

import (
	"net/http"

	"github.com/stackrox/rox/central/logimbue/store"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// New returns a new instance of an HTTP handler for supporting logimbue.
func New(storage store.Store) http.Handler {
	return &handlerImpl{
		storage: storage,
	}
}
