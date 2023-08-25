package debughandler

import (
	"fmt"
	"net/http"
	"os"
	"syscall"

	"github.com/stackrox/rox/pkg/debug"
)

type action struct {
	route       string
	skipConfirm bool
	do          func() error
}

var actions = []action{
	{
		route:       "gc",
		skipConfirm: true,
		do: func() error {
			debug.FreeOSMemory()
			return nil
		},
	},
	{
		route: "quit",
		do: func() error {
			process, err := os.FindProcess(os.Getpid())
			if err != nil {
				return err
			}
			return process.Signal(os.Interrupt)
		},
	},
	{
		route: "quitquitquit",
		do: func() error {
			os.Exit(10)
			return nil
		},
	},
	{
		route: "restart",
		do: func() error {
			return syscall.Exec("/proc/self/exe", os.Args, os.Environ())
		},
	},
}

func (a action) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	_, hasForceParam := req.URL.Query()["force"]
	doIt := req.Method == http.MethodPost || hasForceParam || a.skipConfirm

	if doIt {
		a.serveHTTPAction(w, req)
		return
	}

	a.serveHTTPConfirmDialog(w, req)
}

func (a action) serveHTTPAction(w http.ResponseWriter, _ *http.Request) {
	err := a.do()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (a action) serveHTTPConfirmDialog(w http.ResponseWriter, req *http.Request) {
	pageSrc := fmt.Sprintf(`
<html>
<head>
<style type="text/css">
.doit {
	width: 400px;
	border: 2px solid red;
	background-color: white;
	color: black;
	text-align: center;
	font-size: 20px;
	-webkit-transition-duration: 0.4s;
	transition-duration: 0.4s;
	display: inline-block;
}
.doit:hover {
	background-color: red;
	color:white;
}
</style>
<body>
<form method="post" action="%s">
<button type="submit" class="doit">
Really perform action
<br>
<span style="font-size: 28px; font-face: bold;">%s</span><br>
(you will not see an additional confirmation prompt)
</button>
</form>
</body>
</html>`, req.URL.EscapedPath(), a.route)
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(pageSrc))
}

// Handler returns a handler for debug endpoints.
// PLEASE NOTE: all handlers DO NOT perform any form of authorization or authentication checking. Only expose them on
// an inaccessible endpoint!
func Handler(prefix string) http.Handler {
	mux := http.NewServeMux()
	for _, action := range actions {
		mux.Handle(prefix+"/"+action.route, action)
	}
	return mux
}
