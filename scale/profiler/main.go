package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
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
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           http.FileServer(http.Dir(dir)),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.ListenAndServe()
}
