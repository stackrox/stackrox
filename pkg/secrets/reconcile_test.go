package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	passwordValue = "password"
	tokenValue    = "token"
	tokenPtrValue = "ptrtoken"
)

type config struct {
	Name       string
	Channel    string `scrub:"dependent"`
	OauthToken string `scrub:"always"`
	Map        map[string]string
	ScrubMap   map[string]string `scrub:"map-values"`
}

type toplevel struct {
	Name      string
	Endpoint  string `scrub:"dependent"`
	Username  string `scrub:"dependent"`
	Password  string `scrub:"always"`
	DependInt int    `scrub:"dependent"`
	Config    config
	ConfigPtr *config
	Map       map[string]string
	ScrubMap  map[string]string `scrub:"map-values"`
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

func checkReconciledTopLevel(t *testing.T, obj *toplevel) {
	assert.Equal(t, passwordValue, obj.Password)
	assert.Equal(t, tokenValue, obj.Config.OauthToken)
	assert.Equal(t, tokenPtrValue, obj.ConfigPtr.OauthToken)
}

func checkScrubbedTopLevel(t *testing.T, obj *toplevel) {
	assert.Equal(t, "", obj.Password)
	assert.Equal(t, "", obj.Config.OauthToken)
	assert.Equal(t, "", obj.ConfigPtr.OauthToken)
}

func TestInputErrorsReconcileUpdatedStruct(t *testing.T) {
	var err error
	err = ReconcileScrubbedStructWithExisting(&toplevel{}, nil)
	assert.EqualError(t, err, "invalid input")
	err = ReconcileScrubbedStructWithExisting(&toplevel{}, &config{})
	assert.EqualError(t, err, "type not equal: 'secrets.toplevel' != 'secrets.config'")
	err = ReconcileScrubbedStructWithExisting([]*toplevel{}, []*toplevel{})
	assert.EqualError(t, err, "expected struct, got slice")
}

func TestReconcileUpdatedStruct(t *testing.T) {
	existing := &toplevel{
		Name:      "name",
		Endpoint:  "endpoint",
		Username:  "username",
		DependInt: -1,
		Config:    config{"configName", "channel", tokenValue, nil, nil},
		ConfigPtr: &config{"ptrConfigName", "ptrChannel", tokenPtrValue, nil, nil},
		Password:  passwordValue,
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

	dependentIntUpdate := getScrubbedToplevelClone(existing)
	dependentIntUpdate.DependInt = 42

	cases := []struct {
		name              string
		update            *toplevel
		passes            bool
		skipScrubbedCheck bool
	}{
		{
			"no update, with all creds, fails",
			getTopLevelClone(existing),
			false,
			true,
		},
		{
			"no update, scrubbed creds, passes",
			getScrubbedToplevelClone(existing),
			true,
			true,
		},
		{
			"update cred, fails",
			credUpdate,
			false,
			true,
		},
		{
			"update nested cred, fails",
			credConfigUpdate,
			false,
			true,
		},
		{
			"update nested ptr cred, fails",
			credConfigPtrUpdate,
			false,
			true,
		},
		{
			"update basic field, passes",
			basicUpdate,
			true,
			false,
		},
		{
			"update nested basic field, passes",
			basicConfigUpdate,
			true,
			false,
		},
		{
			"update nested ptr basic field, passes",
			basicConfigPtrUpdate,
			true,
			false,
		},
		{
			"update dependent field, fails",
			dependentUpdate,
			false,
			false,
		},
		{
			"update nested dependent field, fails",
			dependentConfigUpdate,
			false,
			false,
		},
		{
			"update nested ptr dependent field, fails",
			dependentConfigPtrUpdate,
			false,
			false,
		},
		{
			"update non string dependent field, fails",
			dependentIntUpdate,
			false,
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !c.skipScrubbedCheck {
				checkScrubbedTopLevel(t, c.update)
			}
			actual := ReconcileScrubbedStructWithExisting(c.update, existing)
			if c.passes {
				assert.Nilf(t, actual, "Unexpected Error: %s", actual)
				checkReconciledTopLevel(t, c.update)
			} else {
				assert.NotNil(t, actual)
				if !c.skipScrubbedCheck {
					checkScrubbedTopLevel(t, c.update)
				}
			}
		})

	}
}
