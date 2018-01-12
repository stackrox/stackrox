package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// FlattenedConfig is the generic representation of a parsed commandline
type FlattenedConfig map[string]ConfigParams

// Get combines the plural and non plural fields
func (c FlattenedConfig) Get(key string) (ConfigParams, bool) {
	var params ConfigParams
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

func getPID(process string) (int, error) {
	output, err := CombinedOutput("/usr/bin/pgrep", "-f", "-n", process)
	if err != nil {
		if len(output) != 0 {
			return -1, fmt.Errorf("Error getting process '%v'. Output: %v. Err: %+v", process, output, err)
		}
		return -1, fmt.Errorf("Error getting process '%v': %+v", process, err)
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

func parseArg(m FlattenedConfig, arg, nextArg string, commandExpansion map[string]string) bool {
	// If arg containers = then it must be an individual argument and not require the next argument
	// e.g. --security-opt=seccomp as a opposed to --security-opt seccomp
	if strings.Contains(arg, "=") {
		key, value := getKeyValueFromArg(arg)
		expandedKey := getExpandedKey(key, commandExpansion)
		m[expandedKey] = append(m[expandedKey], value)
		return false // Doesn't rely on next argument
	}
	// If the string is a flag and relies on the next value then consolidate
	// e.g. --no-new-privileges true as opposed to --no-new-privileges --selinux-enabled
	if strings.HasPrefix(arg, "-") && !strings.HasPrefix(nextArg, "-") {
		expandedKey := getExpandedKey(arg, commandExpansion)
		m[expandedKey] = append(m[expandedKey], nextArg)
		return true
	}

	// This is the case where the string is standalone like --no-new-privileges
	expandedKey := getExpandedKey(arg, commandExpansion)
	m[expandedKey] = append(m[expandedKey], "")
	return false
}

func parseArgs(m FlattenedConfig, args []string, commandExpansion map[string]string) {
	if len(args) == 0 {
		return
	}
	var skip bool
	for i := 0; i < len(args)-1; i++ {
		if skip {
			skip = !skip
			continue
		}
		skip = parseArg(m, args[i], args[i+1], commandExpansion)
	}
	// Parse last element with empty next arg if skip is not true
	if !skip {
		parseArg(m, args[len(args)-1], "", commandExpansion)
	}
}

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

func getExpandedKey(key string, commandExpansion map[string]string) string {
	if expansion, ok := commandExpansion[key]; ok {
		key = expansion
	}
	return strings.TrimLeft(key, "--")
}

// ConfigParams is a wrapper around the list of values that the docker commandline can have
type ConfigParams []string

// Matches takes a value and Checks the parameter list to see if it contains an exact match
func (d ConfigParams) Matches(value string) bool {
	for _, val := range d {
		if val == value {
			return true
		}
	}
	return false
}

// Contains Checks to see if the parameter list contains the string in one of its elemenets
func (d ConfigParams) Contains(value string) (string, bool) {
	for _, val := range d {
		if strings.Contains(val, value) {
			return val, true
		}
	}
	return "", false
}

// String returns the string version of a list
func (d ConfigParams) String() string {
	return strings.Join(d, " ")
}
