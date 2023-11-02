package command

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stackrox/rox/compliance/collection/file"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/set"
)

var commandsToRetrieve = []string{
	"federation-apiserver",
	"federation-controller-manager",
	"kube-controller-manager",
	"kube-apiserver",
	"etcd",
	"kube-scheduler",
	"kubelet",
}

var flagsWithFiles = set.NewStringSet(
	"kubeconfig",
	"client-ca-file",
	"cni-conf-dir",
	"cni-bin-dir",
	"config",
	"data-dir",
	"tlscacert",
	"tlscert",
	"tlskey",
	"tls-cert-file",
	"tls-private-key-file",
)

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
	cmdLine, err := getCommandLine(processes...)
	if err != nil {
		return nil, true, err
	}
	if cmdLine == "" {
		return nil, false, nil
	}

	processPath, args := getCommandLineArgs(cmdLine)

	// Populate the configuration with the arguments
	a := parseArgs(args)
	return &compliance.CommandLine{
		Process: processPath,
		Args:    a,
	}, true, nil
}

func getProcessFromCmdLineBytes(cmdlineBytes []byte) string {
	if len(cmdlineBytes) == 0 {
		return ""
	}
	processBytes := cmdlineBytes
	index := bytes.Index(cmdlineBytes, []byte("\x00"))
	if index != -1 {
		processBytes = cmdlineBytes[:index]
	}
	return filepath.Base(string(processBytes))
}

func findProcess(process string) (string, error) {
	files, err := os.ReadDir("/host/proc")
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(f.Name())
		if err != nil {
			// This implies it is not a PID
			continue
		}

		cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/host/proc/%d/cmdline", pid))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}

		currProcess := getProcessFromCmdLineBytes(cmdlineBytes)
		if currProcess != process {
			continue
		}
		return string(cmdlineBytes), nil
	}
	return "", nil
}

func getCommandLine(processes ...string) (string, error) {
	for _, p := range processes {
		cmdline, err := findProcess(p)
		if err != nil {
			return "", err
		}
		if cmdline != "" {
			return cmdline, nil
		}
	}
	return "", nil
}

func newArg(k string, values ...string) *compliance.CommandLine_Args {
	k = strings.TrimLeft(k, "-")
	return &compliance.CommandLine_Args{
		Key:    k,
		Values: values,
	}
}

func parseArg(arg, nextArg string) (string, []string, bool) {
	// If arg containers = then it must be an individual argument and not require the next argument
	// e.g. --security-opt=seccomp as a opposed to --security-opt seccomp
	if strings.Contains(arg, "=") {
		k, v := getKeyValueFromArg(arg)
		return k, v, false
	}
	// If the string is a flag and relies on the next value then consolidate
	// e.g. --no-new-privileges true as opposed to --no-new-privileges --selinux-enabled
	if strings.HasPrefix(arg, "-") && !strings.HasPrefix(nextArg, "-") {
		return arg, []string{nextArg}, true
	}

	// This is the case where the string is standalone like --no-new-privileges
	return arg, nil, false
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
		key, values, skip := parseArg(args[i], nextArg)
		if skip {
			i++
		}

		arg := newArg(key, values...)

		// Try to see if key or value is a file path and if so then try to read it and add it to the arg
		if flagsWithFiles.Contains(arg.Key) && len(arg.Values) > 0 {
			f, exists, err := file.EvaluatePath(arg.Values[0], false, true)
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

func getKeyValueFromArg(arg string) (string, []string) {
	argSplit := strings.Split(arg, "=")
	if len(argSplit) == 1 {
		return arg, nil
	}
	values := strings.Split(argSplit[1], ",")
	return argSplit[0], values
}
