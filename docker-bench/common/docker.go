package common

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	//"github.com/docker/docker/daemon"
)

// Config is the docker bench representations of the docker config
type Config map[string]DockerConfigParams

// Get combines the plural and non plural fields
func (c Config) Get(key string) (DockerConfigParams, bool) {
	var params DockerConfigParams
	var found bool
	if foundParams, ok := c[key]; ok {
		params = append(params, foundParams...)
		found = true
	}
	if foundParams, ok := c[key+"s"]; ok {
		params = append(params, foundParams...)
		found = true
	}
	return params, found
}

// DockerConfig is the exported type for benchmarks to reference
var DockerConfig Config
var dockerConfigOnce sync.Once

// DockerClient is the exported docker client for benchmarks to use
var DockerClient *client.Client
var dockerClientOnce sync.Once

// ContainersAll is a slice of all containers in the system
var ContainersAll []types.ContainerJSON

// ContainersRunning is the filtered set of containers that are running
var ContainersRunning []types.ContainerJSON
var containersOnce sync.Once

// Images is the list of images in the system. It does not include all the layers
var Images []types.ImageInspect
var imagesOnce sync.Once

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

func getPID(process string) (int, error) {
	output, err := CombinedOutput("/usr/bin/pgrep", "-f", "-n", process)
	if err != nil {
		return -1, err
	}
	pid, err := strconv.Atoi(output)
	return pid, err
}

func getProcessPID(processNames []string) (pid int, processName string, err error) {
	for _, processName = range processNames {
		pid, err = getPID(processName)
		if err == nil {
			return
		}
	}
	err = fmt.Errorf("Could not find any pids for processes: %+v", processNames)
	return
}

func getCommandLine(pid int) (string, error) {
	cmdline, err := ReadFile(fmt.Sprintf("/proc/%v/cmdline", pid))
	return cmdline, err
}

var commandExpansion = map[string]string{
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

func walkStruct(m map[string]DockerConfigParams, i interface{}) {
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

// Docker's config format is incredibly infuriating as the command line options are different from the config file
func getDockerConfigFromFile(path string, m map[string]DockerConfigParams) error {
	return fmt.Errorf("Docker config file is currently not supported")
	//fileData, err := ReadFile(path)
	//if err != nil {
	//	return err
	//}
	//var config daemon.Config
	//if err := json.Unmarshal([]byte(fileData), &config); err != nil {
	//	return err
	//}
	//walkStruct(m, &config)
	//return nil
}

// DockerConfigParams is a wrapper around the list of values that the docker commandline can have
type DockerConfigParams []string

// Matches takes a value and checks the parameter list to see if it contains an exact match
func (d DockerConfigParams) Matches(value string) bool {
	for _, val := range d {
		if val == value {
			return true
		}
	}
	return false
}

// Contains checks to see if the parameter list contains the string in one of its elemenets
func (d DockerConfigParams) Contains(value string) (string, bool) {
	for _, val := range d {
		if strings.Contains(val, value) {
			return val, true
		}
	}
	return "", false
}

var dockerProcessNames = []string{"docker daemon", "dockerd"}

func nullRune(r rune) bool {
	return r == 0x00
}

func getCommandLineArgs(commandLine string, processName string) []string {
	// Remove the process name from the command line
	// Can't use TrimLeft because /proc/<pid>/cmdline uses NUL char separators
	commandLine = commandLine[len(processName)+1:]
	commandLine = strings.TrimFunc(commandLine, nullRune)

	// Split on the NUL
	args := strings.FieldsFunc(commandLine, nullRune)
	return args
}

func getKeyValueFromArg(arg string) (string, string) {
	argSplit := strings.Split(arg, "=")
	if len(argSplit) == 1 {
		return arg, ""
	}
	return argSplit[0], argSplit[1]
}

func getExpandedKey(key string) string {
	if expansion, ok := commandExpansion[key]; ok {
		key = expansion
	}
	return strings.TrimLeft(key, "--")
}

func parseArg(m Config, arg, nextArg string) bool {
	// If arg containers = then it must be an individual argument and not require the next argument
	// e.g. --security-opt=seccomp as a opposed to --security-opt seccomp
	if strings.Contains(arg, "=") {
		key, value := getKeyValueFromArg(arg)
		expandedKey := getExpandedKey(key)
		m[expandedKey] = append(m[expandedKey], value)
		return false // Doesn't rely on next argument
	}
	// If the string is a flag and relies on the next value then consolidate
	// e.g. --no-new-privileges true as opposed to --no-new-privileges --selinux-enabled
	if strings.HasPrefix(arg, "-") && !strings.HasPrefix(nextArg, "-") {
		expandedKey := getExpandedKey(arg)
		m[expandedKey] = append(m[expandedKey], nextArg)
		return true
	}

	// This is the case where the string is standalone like --no-new-privileges
	expandedKey := getExpandedKey(arg)
	m[expandedKey] = append(m[expandedKey], "")
	return false
}

func parseArgs(m Config, args []string) {
	if len(args) == 0 {
		return
	}
	var skip bool
	for i := 0; i < len(args)-1; i++ {
		if skip {
			skip = !skip
			continue
		}
		skip = parseArg(m, args[i], args[i+1])
	}
	// Parse last element with empty next arg if skip is not true
	if !skip {
		parseArg(m, args[len(args)-1], "")
	}
}

// InitDockerConfig is the Dependency that initializes the docker config
func InitDockerConfig() error {
	var funcErr error
	dockerConfigOnce.Do(func() {
		pid, processName, err := getProcessPID(dockerProcessNames)
		if err != nil {
			funcErr = err
			return
		}

		cmdLine, err := getCommandLine(pid)
		if err != nil {
			funcErr = err
			return
		}
		args := getCommandLineArgs(cmdLine, processName)
		config := make(Config)
		// Populate the configuration with the arguments
		parseArgs(config, args)

		// Add arguments from the config file if it has been passed
		if path, ok := config["config"]; ok {
			if err := getDockerConfigFromFile(path[0], config); err != nil {
				funcErr = err
				return
			}
		}
		DockerConfig = config
		return
	})
	return funcErr
}

// InitDockerClient is the Dependency that initializes the docker client
func InitDockerClient() error {
	var funcErr error
	dockerClientOnce.Do(func() {
		DockerClient, funcErr = client.NewEnvClient()
	})
	return funcErr
}

// GetContainers retrieves the containers and returns running containers, all containers and an error respectively
func GetContainers() ([]types.ContainerJSON, []types.ContainerJSON, error) {
	if err := InitDockerClient(); err != nil {
		return nil, nil, err
	}
	containersList, err := DockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return nil, nil, err

	}
	var containersRunning []types.ContainerJSON
	var containers []types.ContainerJSON

	for _, container := range containersList {
		containerInspect, err := DockerClient.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			return nil, nil, err

		}
		if strings.Contains(containerInspect.Config.Image, "stackrox/docker-bench") {
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
	var funcErr error
	containersOnce.Do(func() {
		runningContainers, allContainers, err := GetContainers()
		if err != nil {
			funcErr = err
			return
		}
		ContainersRunning = runningContainers
		ContainersAll = allContainers
	})
	return funcErr
}

// GetImages returns images and is exported for testing purposes
func GetImages() ([]types.ImageInspect, error) {
	if err := InitDockerClient(); err != nil {
		return nil, err
	}
	imageList, err := DockerClient.ImageList(context.Background(), types.ImageListOptions{All: false})
	if err != nil {
		return nil, err

	}
	var images []types.ImageInspect
	for _, image := range imageList {
		imageInspect, _, err := DockerClient.ImageInspectWithRaw(context.Background(), image.ID)
		if err != nil {
			return nil, err

		}
		images = append(images, imageInspect)
	}
	return images, nil
}

// InitImages initializes the exported Images slice
func InitImages() error {
	var funcErr error
	imagesOnce.Do(func() {
		images, err := GetImages()
		if err != nil {
			funcErr = err
			return
		}
		Images = images
	})
	return funcErr
}
