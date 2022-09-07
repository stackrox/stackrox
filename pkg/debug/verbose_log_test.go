package debug

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ROX12096(t *testing.T) {
	output := ""
	log := func(template string, args ...interface{}) {
		output += fmt.Sprintf(template, args...)
	}
	ROX12096(log, "", "Hello %s!", "World")
	ROX12096(log, "***sac-deploymentnginx-qa***", "This should appear")
	require.Equal(t, output, "This should appear")
}
