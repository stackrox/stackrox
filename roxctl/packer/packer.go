package packer

import (
	"github.com/gobuffalo/packr"
)

const (
	// PropertiesFile contains the command help strings for all commands
	PropertiesFile string = "help.properties"
)

var (
	// RoxctlBox is the packr box for roxctl
	RoxctlBox = packr.NewBox("../help/")
)
