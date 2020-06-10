package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type config struct {
	Name       string
	Channel    string `scrub:"dependent"`
	OauthToken string `scrub:"always"`
}

type toplevel struct {
	Name      string
	Endpoint  string `scrub:"dependent"`
	Username  string `scrub:"dependent"`
	Password  string `scrub:"always"`
	Config    config
	ConfigPtr *config
}

func getTopLevelClone(obj *toplevel) *toplevel {
	configPtrCopy := *obj.ConfigPtr
	objCopy := *obj
	objCopy.ConfigPtr = &configPtrCopy
	return &objCopy
}

func getScrubbedToplevelClone(obj *toplevel) *toplevel {
	scrubbed := getTopLevelClone(obj)
	ScrubSecretsFromStructWithReplacement(scrubbed, "")
	return scrubbed
}

func TestInputErrors(t *testing.T) {
	var err error
	err = ValidateUpdatedStruct(&toplevel{}, nil)
	assert.EqualError(t, err, "invalid input")
	err = ValidateUpdatedStruct(&toplevel{}, &config{})
	assert.EqualError(t, err, "type not equal: 'secrets.toplevel' != 'secrets.config'")
	err = ValidateUpdatedStruct([]*toplevel{}, []*toplevel{})
	assert.EqualError(t, err, "expected struct, got slice")
}

func TestValidateBasicFieldUpdate(t *testing.T) {
	existing := &toplevel{
		Name:      "name",
		Endpoint:  "endpoint",
		Username:  "username",
		Password:  "password",
		Config:    config{"configName", "channel", "token"},
		ConfigPtr: &config{"ptrConfigName", "ptrChannel", "ptrToken"},
	}

	credUpdate := getScrubbedToplevelClone(existing)
	credUpdate.Password = "updatedPassword"

	credConfigUpdate := getScrubbedToplevelClone(existing)
	credConfigUpdate.Config.OauthToken = "updatedToken"

	credConfigPtrUpdate := getScrubbedToplevelClone(existing)
	credConfigPtrUpdate.ConfigPtr.OauthToken = "updatedToken"

	basicUpdate := getScrubbedToplevelClone(existing)
	basicUpdate.Name = "updatedName"

	basicConfigUpdate := getScrubbedToplevelClone(existing)
	basicConfigUpdate.Config.Name = "updatedConfigName"

	basicConfigPtrUpdate := getScrubbedToplevelClone(existing)
	basicConfigPtrUpdate.ConfigPtr.Name = "updatedPtrConfigName"

	dependentUpdate := getScrubbedToplevelClone(existing)
	dependentUpdate.Endpoint = "updatedEndpoint"

	dependentConfigUpdate := getScrubbedToplevelClone(existing)
	dependentConfigUpdate.Config.Channel = "updatedChannel"

	dependentConfigPtrUpdate := getScrubbedToplevelClone(existing)
	dependentConfigPtrUpdate.ConfigPtr.Channel = "updatedPtrChannel"

	cases := []struct {
		name   string
		update *toplevel
		passes bool
	}{
		{
			"no update, with all creds, fails",
			getTopLevelClone(existing),
			false,
		},
		{
			"no update, scrubbed creds, passes",
			getScrubbedToplevelClone(existing),
			true,
		},
		{
			"update cred, fails",
			credUpdate,
			false,
		},
		{
			"update nested cred, fails",
			credConfigUpdate,
			false,
		},
		{
			"update nested ptr cred, fails",
			credConfigPtrUpdate,
			false,
		},
		{
			"update basic field, passes",
			basicUpdate,
			true,
		},
		{
			"update nested basic field, passes",
			basicConfigUpdate,
			true,
		},
		{
			"update nested ptr basic field, passes",
			basicConfigPtrUpdate,
			true,
		},
		{
			"update dependent field, fails",
			dependentUpdate,
			false,
		},
		{
			"update nested dependent field, fails",
			dependentConfigUpdate,
			false,
		},
		{
			"update nested ptr dependent field, fails",
			dependentConfigPtrUpdate,
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := ValidateUpdatedStruct(c.update, existing)
			if c.passes {
				assert.Nilf(t, actual, "Unexpected Error: %s", actual)
			} else {
				assert.NotNil(t, actual)
			}
		})

	}
}
