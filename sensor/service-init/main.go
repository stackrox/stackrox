package main

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	if !features.SensorInstallationExperience.Enabled() {
		log.Fatal("Feature ROX_SENSOR_INSTALLATION_EXPERIENCE is disabled.")
	}

	fmt.Println("Started initContainer")
	for i := 0; i < 500; i++ {
		fmt.Printf("Uptime: %d seconds\n", i)
		time.Sleep(1 * time.Second)
	}
}
