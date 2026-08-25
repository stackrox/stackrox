package main

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/httputil"
)

func main() {
	var (
		port int
		path string
	)
	c := &cobra.Command{
		Use:   "fileserver",
		Short: "runs a generic file server over HTTP",
		RunE: func(*cobra.Command, []string) error {
			return listenAndServe(port, path)
		},
	}

	c.Flags().StringVar(&path, "path", "/pprof", "specify the directory to serve")
	c.Flags().IntVar(&port, "port", 8080, "specify the port to start on the fileserver on")

	if err := c.Execute(); err != nil {
		fmt.Printf("Error running fileserver: %v", err)
	}
}

func listenAndServe(port int, dir string) error {
	server := httputil.NewServer(fmt.Sprintf(":%d", port), http.FileServer(http.Dir(dir)))
	return server.ListenAndServe()
}
