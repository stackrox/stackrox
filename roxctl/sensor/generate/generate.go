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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/defaults"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	cluster = storage.Cluster{
		DynamicConfig: &storage.DynamicClusterConfig{
			AdmissionControllerConfig: &storage.AdmissionControllerConfig{},
		},
	}
	continueIfExists bool
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

func fullClusterCreation(timeout time.Duration) error {
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	service := v1.NewClustersServiceClient(conn)

	id, err := createCluster(service, timeout)
	// If the error is not explicitly AlreadyExists or it is AlreadyExists AND continueIfExists isn't set
	// then return an error

	if err != nil {
		if status.Code(err) == codes.AlreadyExists && continueIfExists {
			// Need to get the clusters and get the one with the name
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			clusterResponse, err := service.GetClusters(ctx, &v1.Empty{})
			if err != nil {
				return errors.Wrap(err, "error getting clusters")
			}
			for _, c := range clusterResponse.GetClusters() {
				if strings.EqualFold(c.GetName(), cluster.GetName()) {
					id = c.GetId()
				}
			}
			if id == "" {
				return fmt.Errorf("error finding preexisting cluster with name %q", cluster.GetName())
			}
		} else {
			return errors.Wrap(err, "error creating cluster")
		}
	}

	if err := getBundle(id, timeout); err != nil {
		return errors.Wrap(err, "error getting cluster zip file")
	}
	return nil
}

// Command defines the deploy command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate creates the required YAML files to deploy StackRox Sensor.",
		Long:  "Generate creates the required YAML files to deploy StackRox Sensor.",
		Run: func(c *cobra.Command, _ []string) {
			_ = c.Help()
		},
	}

	c.PersistentFlags().BoolVar(&continueIfExists, "continue-if-exists", false, "continue with downloading the sensor bundle even if the cluster already exists")
	c.PersistentFlags().StringVar(&cluster.Name, "name", "", "cluster name to identify the cluster")
	c.PersistentFlags().StringVar(&cluster.CentralApiEndpoint, "central", "central.stackrox:443", "endpoint that sensor should connect to")
	c.PersistentFlags().StringVar(&cluster.MainImage, "image", defaults.MainImageRepo(), "image repo sensor should be deployed with")
	c.PersistentFlags().StringVar(&cluster.CollectorImage, "collector-image", "", "image repo collector should be deployed with (leave blank to use default)")
	c.PersistentFlags().StringVar(&cluster.MonitoringEndpoint, "monitoring-endpoint", "", "endpoint for monitoring")
	c.PersistentFlags().BoolVar(&cluster.RuntimeSupport, "runtime", true, "whether or not to have runtime support (DEPRECATED, use Collection Method instead)")
	c.PersistentFlags().BoolVar(&cluster.AdmissionController, "admission-controller", false, "whether or not to use an admission controller for enforcement")

	// Admission controller config
	ac := cluster.DynamicConfig.AdmissionControllerConfig
	c.PersistentFlags().BoolVar(&ac.Enabled, "admission-controller-enabled", false, "dynamic enable for the admission controller")
	c.PersistentFlags().Int32Var(&ac.TimeoutSeconds, "admission-controller-timeout", 3, "timeout in seconds for the admission controller")
	c.PersistentFlags().BoolVar(&ac.ScanInline, "admission-controller-scan-inline", false, "get scans inline when using the admission controller")
	c.PersistentFlags().BoolVar(&ac.DisableBypass, "admission-controller-disable-bypass", false, "disable the bypass annotations for the admission controller")

	c.PersistentFlags().Var(&collectionTypeWrapper{CollectionMethod: &cluster.CollectionMethod}, "collection-method", "which collection method to use for runtime support (none, kernel-module, ebpf)")
	c.AddCommand(k8s())
	return c
}

func createCluster(svc v1.ClustersServiceClient, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// Call detection and return the returned alerts.
	response, err := svc.PostCluster(ctx, &cluster)
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
		return errors.Wrapf(err, "Unable to create folder %q", outputFolder)
	}

	for _, f := range reader.File {
		fileReader, err := f.Open()
		if err != nil {
			return errors.Wrapf(err, "Unable to open file %q", f.Name)
		}
		data, err := ioutil.ReadAll(fileReader)
		if err != nil {
			return errors.Wrapf(err, "Unable to read file %q", f.Name)
		}
		if err := ioutil.WriteFile(filepath.Join(outputFolder, f.Name), data, f.Mode()); err != nil {
			return errors.Wrapf(err, "Unable to write file %q", f.Name)
		}
	}
	printf("Successfully wrote sensor folder %q\n", outputFolder)
	return nil
}

func getBundle(id string, timeout time.Duration) error {
	url := common.GetURL("/api/extensions/clusters/zip")
	client := common.GetHTTPClient(timeout)
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
	defer func() {
		_ = resp.Body.Close()
	}()
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
			return errors.Wrap(err, "Error writing out zip file")
		}
		printf("Successfully wrote sensor zip file\n")
	} else {
		file, err := os.Create(outputFilename)
		if err != nil {
			return errors.Wrapf(err, "Could not create file %q", outputFilename)
		}
		if _, err := io.Copy(file, resp.Body); err != nil {
			_ = file.Close()
			return errors.Wrap(err, "Error writing out zip file")
		}
		if err := file.Close(); err != nil {
			return errors.Wrap(err, "Error closing file")
		}
		if err := writeZipToFolder(outputFilename); err != nil {
			return err
		}
	}
	return nil
}
