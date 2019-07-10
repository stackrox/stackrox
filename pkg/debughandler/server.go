package debughandler

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/utils"
)

// StartServer starts a debug handler server. This function always returns with a non-nil error code.
func StartServer(port string) error {
	if !devbuild.IsEnabled() {
		return errors.New("a debug server can only be started for development builds")
	}

	if port == "" {
		port = "9999"
	}

	addr := fmt.Sprintf("127.0.0.1:%s", port)
	return http.ListenAndServe(addr, Handler(""))
}

// MustStartServerAsync starts a debug handler server in a goroutine. If the server exits, the program panics.
func MustStartServerAsync(port string) {
	go func() {
		utils.Must(StartServer(port))
	}()
}
