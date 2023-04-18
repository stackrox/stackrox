package flags

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
)

const (
	noColorName = "no-color"
	noColorFlag = "--" + noColorName
)

// AddNoColor adds the noColor flag to the base command.
func AddNoColor(c *cobra.Command) {
	// We don't care about this value since we need to check args if it contains --no-color flag
	// because printers are initialized before all arguments are parsed.
	// Printer is required to initialize commands thus we cannot follow
	// https://github.com/fatih/color/blob/v1.13.0/doc.go#L109-L119
	var noColor bool
	c.PersistentFlags().BoolVar(&noColor, noColorName, false, "Disable color output. Alternately disable the color output by setting the ROX_NO_COLOR_PRINTER environment variable")
}

// HasNoColor returns true is passed args contain noColorFlag
func HasNoColor(args []string) bool {
	for _, arg := range args {
		if arg == noColorFlag {
			return true
		}
	}
	if env.NoColorPrinterEnv.BooleanSetting() != env.NoColorPrinterEnv.DefaultBooleanSetting() {
		return env.NoColorPrinterEnv.BooleanSetting()
	}
	return false
}
