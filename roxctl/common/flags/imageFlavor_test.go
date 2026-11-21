package flags

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stretchr/testify/require"
)

func TestAddImageDefaultsUsesOpenSourceByDefault(t *testing.T) {
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	var imageFlavor string

	AddImageDefaults(flagSet, &imageFlavor)

	flag := flagSet.Lookup(ImageDefaultsFlagName)
	require.NotNil(t, flag)
	require.Equal(t, defaults.ImageFlavorNameOpenSource, flag.DefValue)
	require.Equal(t, defaults.ImageFlavorNameOpenSource, imageFlavor)
}
