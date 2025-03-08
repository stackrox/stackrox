package flags

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestFlagOrSettingValue(t *testing.T) {
	// 1. Default, unchanged flag value and setting not set should lead to the default value being returned.
	cmd := &cobra.Command{}

	addCentralAuthFlags(cmd)

	assert.Empty(t, Password())

	// 2. Change the flag value. The changed flag value should be returned, irrespective of whether the setting is set.
	t.Setenv("ROX_ADMIN_PASSWORD", "some-test-value")
	cmd = &cobra.Command{}
	addCentralAuthFlags(cmd)
	assert.NoError(t, cmd.ParseFlags([]string{"--password", "some-other-test-value"}))
	assert.Equal(t, "some-other-test-value", Password())

	// 3. Default flag value and setting's value set should return the settings value instead.
	t.Setenv("ROX_ADMIN_PASSWORD", "some-test-value")
	cmd = &cobra.Command{}
	authFlagSet.Lookup("password").Changed = false
	addCentralAuthFlags(cmd)
	assert.Equal(t, "some-test-value", Password())
}

func TestBooleanFlagOrSettingValue(t *testing.T) {
	// 1. Default, unchanged flag value and setting not set should lead to the default value being returned.
	cmd := &cobra.Command{}

	AddCentralConnectionFlags(cmd)

	assert.False(t, UseInsecure())

	// 2. Change the flag value. The changed flag value should be returned, irrespective of whether the setting is set.
	t.Setenv("ROX_INSECURE_CLIENT", "true")
	cmd = &cobra.Command{}
	AddCentralConnectionFlags(cmd)
	assert.NoError(t, cmd.ParseFlags([]string{"--insecure=false"}))
	assert.Equal(t, false, UseInsecure())

	// 3. Default flag value and setting's value set should return the settings value instead.
	t.Setenv("ROX_INSECURE_CLIENT", "true")
	cmd = &cobra.Command{}
	connectionFlags.Lookup("insecure").Changed = false
	AddCentralConnectionFlags(cmd)
	assert.Equal(t, true, UseInsecure())
}
