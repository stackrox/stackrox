package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// ReadFile takes in a filename and returns the body in string form or an error
func ReadFile(filename string) (string, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}

// CombinedOutput is a helper function to exec.Command where the combined output is returned in string form
func CombinedOutput(cmd string, args ...string) (string, error) {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

var possibleSystemdPaths = []string{
	"/etc/systemd/system/%v",
	"/lib/systemd/system/%v",
	"/usr/lib/systemd/system/%v",
}

// GetSystemdFile finds the systemd file for a particular service
func GetSystemdFile(service string) string {
	for _, template := range possibleSystemdPaths {
		path := fmt.Sprintf(template, service)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "systemd path not found"
}
