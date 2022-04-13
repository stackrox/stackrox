package utils

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"github.com/stackrox/stackrox/pkg/set"
)

const hostProc = "/host/proc"

func findProcessArgs(processes set.StringSet) ([]string, error) {
	files, err := os.ReadDir(hostProc)
	if err != nil {
		return nil, errors.Wrap(err, "could not read host proc")
	}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(f.Name()); err != nil {
			// This implies it is not a PID
			continue
		}

		cmdlineFile := filepath.Join(hostProc, f.Name(), "cmdline")
		cmdlineBytes, err := os.ReadFile(cmdlineFile)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, errors.Wrapf(err, "error reading file %s", cmdlineFile)
		}

		fields := strings.FieldsFunc(string(cmdlineBytes), nullRune)
		if len(fields) == 0 {
			continue
		}
		if !processes.Contains(fields[0]) {
			continue
		}
		return fields[1:], nil
	}
	return nil, nil
}

func nullRune(r rune) bool {
	return r == 0x00
}

// NewFlagSet returns a new set for the passes process
func NewFlagSet(process string) *flag.FlagSet {
	cmdline := flag.NewFlagSet(process, flag.ContinueOnError)
	cmdline.ParseErrorsWhitelist.UnknownFlags = true
	return cmdline
}

// ParseFlags takes in a a set of processes that may match and tries to parse the flags successfully
func ParseFlags(processes set.StringSet, pflag *flag.FlagSet) error {
	args, err := findProcessArgs(processes)
	if err != nil {
		return errors.Wrapf(err, "error finding arguments for %+v", processes.AsSlice())
	}
	if err := pflag.Parse(args); err != nil {
		return errors.Wrapf(err, "error parsing args: %+v", args)
	}
	return nil
}
