package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/stackrox/rox/pkg/docker"
)

// DockerConfig is the exported type for benchmarks to reference
var DockerConfig FlattenedConfig
var dockerConfigOnce sync.Once
var dockerConfigErr error

// DockerClient is the exported docker client for benchmarks to use
var DockerClient *client.Client
var dockerClientOnce sync.Once
var dockerClientErr error

// ContainersAll is a slice of all containers in the system
var ContainersAll []types.ContainerJSON

// ContainersRunning is the filtered set of containers that are running
var ContainersRunning []types.ContainerJSON
var containersOnce sync.Once
var containerErr error

// Images is the list of images in the system. It does not include all the layers
var Images []types.ImageInspect
var imagesOnce sync.Once
var imageErr error

// DockerInfo contains the info of the docker daemon
var DockerInfo types.Info
var infoOnce sync.Once
var infoErr error

// GetReadableImageName takes in a docker image and returns the human readable repo:tag combination or the ID if
// the tag doesn't exist
func GetReadableImageName(image types.ImageInspect) string {
	if len(image.RepoTags) != 0 {
		return image.RepoTags[0]
	}
	if len(image.RepoDigests) != 0 {
		return image.RepoDigests[0]
	}
	return image.ID
}

var dockerCommandExpansion = map[string]string{
	"-b": "--bridge",
	"-D": "--debug",
	"-G": "--group",
	"-H": "--host",
	"-l": "--log-level",
	"-p": "--pidfile",
	"-s": "--storage-driver",
}

func getTagValue(tag string) (string, bool) {
	tag = strings.TrimSuffix(tag, ",omitempty")
	return tag, tag != "-" && tag != ""
}

func walkStruct(m map[string]ConfigParams, i interface{}) {
	val := reflect.ValueOf(i)
	if reflect.TypeOf(i).Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}
	valType := reflect.TypeOf(val.Interface())
	for i := 0; i < val.NumField(); i++ {
		typeField := valType.Field(i)
		field := val.Field(i)
		tagStr, valid := getTagValue(string(typeField.Tag.Get("json")))
		if !valid && typeField.Type.Kind() != reflect.Struct {
			continue
		}
		switch typeField.Type.Kind() {
		case reflect.String:
			m[tagStr] = append(m[tagStr], field.Interface().(string))
		case reflect.Struct:
			if field.CanInterface() {
				walkStruct(m, field.Interface())
			}
		case reflect.Slice:
			strSlice, ok := field.Interface().([]string)
			if !ok {
				continue
			}
			m[tagStr] = append(m[tagStr], strSlice...)
		case reflect.Ptr:
			if field.IsNil() {
				continue
			}
			m[tagStr] = append(m[tagStr], fmt.Sprintf("%v", field.Elem().Interface()))
		case reflect.Map:
			if field.IsNil() {
				continue
			}
			stringMap, ok := field.Interface().(map[string]string)
			if ok {
				for k, v := range stringMap {
					m[tagStr] = append(m[tagStr], fmt.Sprintf("%v=%v", k, v))
				}
				continue
			}
		default:
			m[tagStr] = append(m[tagStr], fmt.Sprintf("%v", field.Interface()))
		}
	}
}

func appendToConfig(m map[string]ConfigParams, key, value string) {
	m[key] = append(m[key], value)
}

func boolToString(b bool) string {
	return fmt.Sprintf("%v", b)
}

// Docker's config format is incredibly infuriating as the command line options are different from the config file
func getDockerConfigFromFile(path string, m map[string]ConfigParams) error {
	fileData, err := ReadFile(path)
	if err != nil {
		return err
	}
	var config Config
	if err := json.Unmarshal([]byte(fileData), &config); err != nil {
		return err
	}
	// Populate most fields automatically
	walkStruct(m, &config)
	return nil
}

var dockerProcessNames = []string{"docker daemon", "dockerd"}

// InitDockerConfig is the Dependency that initializes the docker config
func InitDockerConfig() error {
	dockerConfigOnce.Do(func() {
		pid, processName, err := getProcessPID(dockerProcessNames)
		if err != nil {
			dockerConfigErr = err
			return
		}

		cmdLine, err := getCommandLine(pid)
		if err != nil {
			dockerConfigErr = err
			return
		}
		args := getCommandLineArgs(cmdLine, processName)
		config := make(FlattenedConfig)
		// Populate the configuration with the arguments
		parseArgs(config, args, dockerCommandExpansion)

		// Add arguments from the config file if it has been passed
		if path, ok := config["config"]; ok {
			if err := getDockerConfigFromFile(path[0], config); err != nil {
				dockerConfigErr = err
				return
			}
		}
		DockerConfig = config
		return
	})
	return dockerConfigErr
}

// InitDockerClient is the Dependency that initializes the docker client
func InitDockerClient() error {
	dockerClientOnce.Do(func() {
		DockerClient, dockerClientErr = docker.NewClient()
	})
	return dockerClientErr
}

// GetContainers retrieves the containers and returns running containers, all containers and an error respectively
func GetContainers() ([]types.ContainerJSON, []types.ContainerJSON, error) {
	if err := InitDockerClient(); err != nil {
		return nil, nil, err
	}
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	containersList, err := DockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, nil, err

	}
	var containersRunning []types.ContainerJSON
	var containers []types.ContainerJSON

	for _, container := range containersList {
		ctx, cancel := docker.TimeoutContext()
		defer cancel()
		containerInspect, err := DockerClient.ContainerInspect(ctx, container.ID)
		if err != nil {
			return nil, nil, err

		}
		if strings.Contains(containerInspect.Config.Image, "stackrox/main") {
			continue
		}
		if containerInspect.State.Status == "running" {
			containersRunning = append(containersRunning, containerInspect)
		}
		containers = append(containers, containerInspect)
	}
	return containersRunning, containers, err
}

// InitContainers initializes ContainersRunning and ContainersAll
func InitContainers() error {
	containersOnce.Do(func() {
		runningContainers, allContainers, err := GetContainers()
		if err != nil {
			containerErr = err
			return
		}
		ContainersRunning = runningContainers
		ContainersAll = allContainers
	})
	return containerErr
}

// GetImages returns images and is exported for testing purposes
func GetImages() ([]types.ImageInspect, error) {
	if err := InitDockerClient(); err != nil {
		return nil, err
	}
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	imageList, err := DockerClient.ImageList(ctx, types.ImageListOptions{All: false})
	if err != nil {
		return nil, err

	}
	var images []types.ImageInspect
	for _, image := range imageList {
		ctx, cancel := docker.TimeoutContext()
		defer cancel()
		imageInspect, _, err := DockerClient.ImageInspectWithRaw(ctx, image.ID)
		if err != nil {
			return nil, err

		}
		images = append(images, imageInspect)
	}
	return images, nil
}

// InitImages initializes the exported Images slice
func InitImages() error {
	imagesOnce.Do(func() {
		images, err := GetImages()
		if err != nil {
			imageErr = err
			return
		}
		Images = images
	})
	return imageErr
}

// InitInfo initializes the docker info
func InitInfo() error {
	infoOnce.Do(func() {
		if err := InitDockerClient(); err != nil {
			infoErr = err
		}
		ctx, cancel := docker.TimeoutContext()
		defer cancel()
		info, err := DockerClient.Info(ctx)
		if err != nil {
			infoErr = err
			return
		}
		DockerInfo = info
	})
	return infoErr
}
