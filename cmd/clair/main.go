package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	imagefmt "bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/urlfmt"
	"github.com/spf13/cobra"
)

const (
	dockerConfigPath = "/config/config.json"
)

var (
	processors   map[string]*scanProcessor
	registryAuth map[string]*basicAuth
	clair        *clairClient

	registryOverride = map[string]string{
		"docker.io": "registry-1.docker.io",
	}

	fullyQualifiedRegistryOverride = map[string]string{
		"docker.io": "https://registry-1.docker.io",
	}

	log = logging.LoggerForModule()
)

func init() {
	processors = make(map[string]*scanProcessor)
}

type config struct {
	clairEndpoint    string
	dockerConfigPath string
	image            string
	preventEndpoint  string
}

func cmd() *cobra.Command {
	var cfg config
	c := &cobra.Command{
		Use:   "run",
		Short: "Scan pushes images to a Clair instance",
		Long:  "Scan pushes images to a Clair instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfg)
		},
	}
	c.Flags().StringVarP(&cfg.clairEndpoint, "clair", "e", "127.0.0.1:6060", "clair endpoint")
	c.Flags().StringVarP(&cfg.dockerConfigPath, "config", "c", dockerConfigPath, "docker config path")
	c.Flags().StringVarP(&cfg.image, "image", "i", "", "image name to run")
	c.Flags().StringVarP(&cfg.preventEndpoint, "prevent", "m", os.Getenv("LOCAL_API_ENDPOINT"), "Prevent endpoint for automatic parsing")
	return c
}

func stripHTTPPrefix(s string) string {
	s = strings.TrimPrefix(s, "https://")
	return strings.TrimPrefix(s, "http://")
}

func evaluateRegistryOverride() {
	overrides, ok := os.LookupEnv("PREVENT_REGISTRY_OVERRIDE")
	if !ok {
		return
	}
	csv := strings.Split(overrides, ",")
	for _, c := range csv {
		spl := strings.SplitN(c, "=", 2)
		if len(spl) != 2 {
			log.Fatalf("Environment variable section PREVENT_REGISTRY_OVERRIDE '%v' must be separated with an = sign", c)
		}
		fullyQualifiedRegistryOverride[spl[0]] = spl[1]
		registryOverride[stripHTTPPrefix(spl[0])] = stripHTTPPrefix(spl[1])
	}
}

// This is mostly to avoid Mac issues due to osxkeychain
// e.g. PREVENT_REGISTRY_AUTH=registry-1.docker.io=<base64 encoded>
func overrideRegistryAuth() {
	overrides, ok := os.LookupEnv("PREVENT_REGISTRY_AUTH")
	if !ok {
		return
	}
	csv := strings.Split(overrides, ",")
	for _, c := range csv {
		spl := strings.SplitN(c, "=", 2)
		if len(spl) != 2 {
			log.Fatalf("Environment variable section PREVENT_REGISTRY_AUTH '%v' must be separated with an = sign", c)
		}
		registry := strings.TrimSpace(spl[0])

		sDec, err := base64.URLEncoding.DecodeString(spl[1])
		if err != nil {
			log.Fatalf("Could not base64 decode '%v'", spl[1])
		}

		authSpl := strings.SplitN(string(sDec), ":", 2)
		if len(spl) != 2 {
			log.Fatalf("Could not parse registry auth from environment variable for %v", registry)
		}
		registryAuth[registry] = &basicAuth{
			username: strings.TrimSpace(authSpl[0]),
			password: strings.TrimSpace(authSpl[1]),
		}
	}
}

func populateRegistryAuth(file string) {
	var err error
	registryAuth, err = readDockerConfig(file)
	if err != nil {
		log.Errorf("Unable to read docker config: %s", err)
		registryAuth = make(map[string]*basicAuth)
	}
}

func main() {
	if err := cmd().Execute(); err != nil {
		log.Errorf("unable to execute: %s", err)
	}
}

func run(cfg config) error {
	if cfg.clairEndpoint == "" {
		return errors.New("Endpoint for Clair must be defined")
	}
	if cfg.image == "" && cfg.preventEndpoint == "" {
		return errors.New("Either image or prevent must be defined")
	}
	endpoint, err := urlfmt.FormatURL(cfg.clairEndpoint, false, false)
	if err != nil {
		log.Fatalf("Could not parse Clair endpoint %v: %v", endpoint, err)
	}
	clair = &clairClient{endpoint: endpoint}

	populateRegistryAuth(cfg.dockerConfigPath)
	evaluateRegistryOverride()
	overrideRegistryAuth()

	if cfg.image != "" {
		// Parse Image
		protoImage := imagefmt.GenerateImageFromString(cfg.image)
		return runImage(protoImage)
	}

	// Go get images, check that they are authenticated, then add them to clair
	images, err := getImages(cfg.preventEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	if !hasNecessaryAuth(images) {
		log.Fatalf("Not properly logged in for all registries. Please see the logs above")
	}
	for _, image := range images {
		log.Infof("Processing image '%v'", imagefmt.Wrapper{Image: image})
		if err := runImage(image); err != nil {
			log.Errorf("Error analyzing image %v: %+v", imagefmt.Wrapper{Image: image}, err)
		}
	}
	return nil
}

func runImage(image *v1.Image) error {
	registryEndpoint := image.GetName().GetRegistry()
	if endpoint, ok := fullyQualifiedRegistryOverride[image.GetName().GetRegistry()]; ok {
		registryEndpoint = endpoint
	}
	registryID := image.GetName().GetRegistry()
	if id, ok := registryOverride[image.GetName().GetRegistry()]; ok {
		registryID = id
	}

	auth, ok := registryAuth[registryID]
	if !ok {
		if image.GetName().GetRegistry() != "docker.io" {
			return fmt.Errorf("No registry auth for '%v' found. Please docker login to %v", image.GetName().GetRegistry(), image.GetName().GetRegistry())
		}
		auth = &basicAuth{}
	}

	// See of the processor for the registry has been initialized
	processor, ok := processors[registryID]
	if !ok {
		var err error
		processor, err = newProcessor(registryEndpoint, auth, clair)
		if err != nil {
			return err
		}
	}
	return processor.processImage(image)
}
