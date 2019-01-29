package command

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"

	"github.com/stackrox/rox/compliance/collection/file"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

var commandsToRetrieve = []string{
	"dockerd",

	"federation-apiserver",
	"federation-controller-manager",
	"kube-apiserver",
	"etcd",
	"kube-scheduler",
	"kubelet",
}

// RetrieveCommands returns the commandlines of the services to be evaluated
func RetrieveCommands() (map[string]*compliance.CommandLine, error) {
	commands := make(map[string]*compliance.CommandLine)
	for _, c := range commandsToRetrieve {
		c, exists, err := parseCommandline(c)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		commands[c.GetProcess()] = c
	}
	return commands, nil
}

func parseCommandline(processes ...string) (*compliance.CommandLine, bool, error) {
	pid, err := getProcessPID(processes)
	if err != nil {
		// This means we couldn't find the pid so return that it doesn't exist
		return nil, false, nil
	}
	cmdLine, err := getCommandLine(pid)
	if err != nil {
		return nil, true, err
	}

	processPath, args := getCommandLineArgs(cmdLine)

	// Populate the configuration with the arguments
	a := parseArgs(args)
	return &compliance.CommandLine{
		Process: processPath,
		Args:    a,
	}, true, nil
}

func getPID(process string) (int, error) {
	output, err := exec.Command("/usr/bin/pgrep", "--exact", process).CombinedOutput()
	if err != nil {
		if len(output) != 0 {
			return -1, fmt.Errorf("Error getting process %q. Output: %s. Err: %v", process, output, err)
		}
		return -1, fmt.Errorf("Error getting process %q: %v", process, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(output)))
	return pid, err
}

func getProcessPID(processNames []string) (int, error) {
	for _, processName := range processNames {
		if pid, err := getPID(processName); err == nil {
			return pid, nil
		}
	}
	return 0, fmt.Errorf("Could not find any pids for processes: %+v", processNames)
}

func getCommandLine(pid int) (string, error) {
	cmdline, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", err
	}
	return string(cmdline), err
}

func newArg(k, v string) *compliance.CommandLine_Args {
	k = strings.TrimLeft(k, "--")
	k = strings.TrimLeft(k, "-")
	return &compliance.CommandLine_Args{
		Key:   k,
		Value: v,
	}
}

func parseArg(arg, nextArg string) (string, string, bool) {
	// If arg containers = then it must be an individual argument and not require the next argument
	// e.g. --security-opt=seccomp as a opposed to --security-opt seccomp
	if strings.Contains(arg, "=") {
		k, v := getKeyValueFromArg(arg)
		return k, v, false
	}
	// If the string is a flag and relies on the next value then consolidate
	// e.g. --no-new-privileges true as opposed to --no-new-privileges --selinux-enabled
	if strings.HasPrefix(arg, "-") && !strings.HasPrefix(nextArg, "-") {
		return arg, nextArg, true
	}

	// This is the case where the string is standalone like --no-new-privileges
	return arg, "", false
}

func parseArgs(args []string) []*compliance.CommandLine_Args {
	if len(args) == 0 {
		return nil
	}

	var retArgs []*compliance.CommandLine_Args
	for i := 0; i < len(args); i++ {
		var nextArg string
		if i+1 < len(args) {
			nextArg = args[i+1]
		}
		key, value, skip := parseArg(args[i], nextArg)
		if skip {
			i++
		}
		// Try to see if key or value is a file path and if so then try to read it and add it to the arg
		arg := newArg(key, value)

		if strings.HasPrefix(key, "/") {
			f, exists, err := file.EvaluatePath(key, true)
			if exists && err == nil {
				arg.File = f
			}
		}
		if strings.HasPrefix(value, "/") {
			f, exists, err := file.EvaluatePath(value, true)
			if exists && err == nil {
				arg.File = f
			}
		}
		retArgs = append(retArgs, arg)
	}
	return retArgs
}

func nullRune(r rune) bool {
	return r == 0x00
}

func getCommandLineArgs(commandLine string) (string, []string) {
	// Split on the NUL
	args := strings.FieldsFunc(commandLine, nullRune)
	if len(args) == 1 {
		return commandLine, nil
	}
	return args[0], args[1:]
}

func getKeyValueFromArg(arg string) (string, string) {
	argSplit := strings.Split(arg, "=")
	if len(argSplit) == 1 {
		return arg, ""
	}
	return argSplit[0], argSplit[1]
}
