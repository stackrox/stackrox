package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/safe"
)

var (
	log = logging.LoggerForModule()
)

const (
	alternativeCAPath = `/run/secrets/stackrox.io/ca/ca.pem`
)

func main() {
	clientconn.SetUserAgent(clientconn.AdmissionController)
	ctx := context.Background()

	if err := safe.RunE(func() error {
		if err := configureCA(); err != nil {
			return err
		}
		if err := configureCerts("default"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Errorf("Failed to configure certificates: %v. Connection to sensor might fail.", err)
	}

	sensorConn, err := clientconn.AuthenticatedGRPCConnection(ctx, "sensor.default.svc:443", mtls.SensorSubject)

	if err != nil {
		log.Errorf("Could not establish a gRPC connection to Sensor: %v.")
	}

	// sensorConn.WaitForStateChange(ctx, connectivity.Ready)

	imageClient := sensor.NewImageServiceClient(sensorConn)
	resp, err := imageClient.GetImage(ctx, &sensor.GetImageRequest{
		Namespace: "default",
		Image: &storage.ContainerImage{
			Name: &storage.ImageName{
				FullName: "localhost:5001/stackrox/stackrox:latest",
			},
		},
	})

	if err != nil {
		log.Errorf("Failed to get image: %v", err)
		return
	}

	log.Infof("Response: %s", resp.Image.Name)
	log.Infof("Scan: %d", resp.Image.Scan)
	if resp.Image.Scan != nil {
		log.Infof("Component count: %d", len(resp.Image.Scan.Components))
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	vm := &storage.VirtualMachine{
		Id: hostname,
	}

	if err := populateComponentList(vm); err != nil {
		log.Errorf("Failed to populate component list: %v", err)
	}

	vmClient := sensor.NewVirtualMachineServiceClient(sensorConn)
	vmResp, err := vmClient.UpsertVirtualMachine(ctx, &sensor.UpsertVirtualMachineRequest{VirtualMachine: vm})

	if err != nil {
		log.Errorf("Failed to upsert VM: %v", err)
		return
	}

	log.Infof("VM upsert success: %v", vmResp.Success)
}

func populateComponentList(vm *storage.VirtualMachine) error {
	// Run RPM command to fetch list of installed RPMs in JSON format
	cmd := exec.Command("rpm", "-qa", "--qf", `{"name":"%{NAME}","version":"%{VERSION}","release":"%{RELEASE}","arch":"%{ARCH}"\}\n`)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute rpm command: %v", err)
	}

	// Parse the JSON output and populate components
	components := []*storage.EmbeddedImageScanComponent{}
	bufferScanner := bufio.NewScanner(bytes.NewReader(output))
	for bufferScanner.Scan() {
		var rpmInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Release string `json:"release"`
			Arch    string `json:"arch"`
		}
		if err := json.Unmarshal(bufferScanner.Bytes(), &rpmInfo); err != nil {
			return fmt.Errorf("failed to parse rpm output: %v", err)
		}

		component := &storage.EmbeddedImageScanComponent{
			Name:         rpmInfo.Name,
			Version:      fmt.Sprintf("%s-%s", rpmInfo.Version, rpmInfo.Release),
			Source:       storage.SourceType_OS,
			Architecture: rpmInfo.Arch,
		}
		components = append(components, component)
	}

	if err := bufferScanner.Err(); err != nil {
		return fmt.Errorf("error reading rpm output: %v", err)
	}

	log.Infof("Found %d components", len(components))
	vm.Scan = &storage.VirtualMachineScan{
		Components: components,
	}
	return nil
}
