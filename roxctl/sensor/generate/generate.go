package generate

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/defaults"
)

const (
	connectionTimeout = 5 * time.Second
)

var (
	cluster storage.Cluster
)

type zipPost struct {
	ID string `json:"id"`
}

func printf(val string, args ...interface{}) {
	if docker.IsContainerized() {
		fmt.Fprintf(os.Stderr, val, args...)
	} else {
		fmt.Printf(val, args...)
	}
}

func fullClusterCreation() error {
	id, err := createCluster()
	if err != nil {
		return fmt.Errorf("Error creating cluster: %v", err)
	}
	if err := getBundle(id); err != nil {
		return fmt.Errorf("Error getting cluster zip file: %v", err)
	}
	return nil
}

// Command defines the deploy command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate creates the required YAML files to deploy StackRox Central.",
		Long:  "Generate creates the required YAML files to deploy StackRox Central.",
		Run: func(c *cobra.Command, _ []string) {
			c.Help()
		},
	}

	c.PersistentFlags().StringVar(&cluster.Name, "name", "", "cluster name to identify the cluster")
	c.PersistentFlags().StringVar(&cluster.CentralApiEndpoint, "central", "central.stackrox:443", "endpoint that sensor should connect to")
	c.PersistentFlags().StringVar(&cluster.MainImage, "image", defaults.MainImage, "image sensor should be deployed with")
	c.PersistentFlags().StringVar(&cluster.MonitoringEndpoint, "monitoring-endpoint", "", "endpoint for monitoring")
	c.PersistentFlags().BoolVar(&cluster.RuntimeSupport, "runtime", true, "whether or not to have runtime support")
	c.AddCommand(k8s())
	return c
}

func createCluster() (string, error) {
	// Create the connection to the central detection service.
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return "", err
	}
	service := v1.NewClustersServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	// Call detection and return the returned alerts.
	response, err := service.PostCluster(ctx, &cluster)
	if err != nil {
		return "", err
	}
	return response.GetCluster().GetId(), nil
}

func parseFilenameFromHeader(header http.Header) (string, error) {
	data := header.Get("Content-Disposition")
	if data == "" {
		return data, fmt.Errorf("could not parse filename from header: %+v", header)
	}
	data = strings.TrimPrefix(data, "attachment; filename=")
	return strings.Trim(data, `"`), nil
}

func writeZipToFolder(zipName string) error {
	reader, err := zip.OpenReader(zipName)
	if err != nil {
		return err
	}

	outputFolder := strings.TrimRight(zipName, filepath.Ext(zipName))
	if err := os.MkdirAll(outputFolder, 0777); err != nil {
		return fmt.Errorf("Unable to create folder %q: %v", outputFolder, err)
	}

	for _, f := range reader.File {
		fileReader, err := f.Open()
		if err != nil {
			return fmt.Errorf("Unable to open file %q: %v", f.Name, err)
		}
		data, err := ioutil.ReadAll(fileReader)
		if err != nil {
			return fmt.Errorf("Unable to read file %q: %v", f.Name, err)
		}
		if err := ioutil.WriteFile(filepath.Join(outputFolder, f.Name), data, f.Mode()); err != nil {
			return fmt.Errorf("Unable to write file %q: %v", f.Name, err)
		}
	}
	printf("Successfully wrote sensor folder %q\n", outputFolder)
	return nil
}

func getBundle(id string) error {
	url := common.GetURL("/api/extensions/clusters/zip")
	client := common.GetHTTPClient(connectionTimeout)
	body, _ := json.Marshal(&zipPost{ID: id})
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	common.AddAuthToRequest(req)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Expected status code 200, but received %d", resp.StatusCode)
		}
		return fmt.Errorf("Expected status code 200, but received %d. Response Body: %s", resp.StatusCode, string(data))

	}

	outputFilename, err := parseFilenameFromHeader(resp.Header)
	if err != nil {
		return err
	}
	// If containerized, then write a zip file
	if docker.IsContainerized() {
		if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
			return fmt.Errorf("Error writing out zip file: %v", err)
		}
		printf("Successfully wrote sensor zip file\n")
	} else {
		file, err := os.Create(outputFilename)
		if err != nil {
			return fmt.Errorf("Could not create file %q: %v", outputFilename, err)
		}
		if _, err := io.Copy(file, resp.Body); err != nil {
			file.Close()
			return fmt.Errorf("Error writing out zip file: %v", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("Error closing file: %v", err)
		}
		if err := writeZipToFolder(outputFilename); err != nil {
			return err
		}
	}
	return nil
}
