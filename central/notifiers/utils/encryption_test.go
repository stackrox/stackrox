package utils

import (
	"encoding/base64"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	pkgNotifiers "github.com/stackrox/rox/pkg/notifiers"
	"github.com/stretchr/testify/suite"
)

func TestNotifierSecurity(t *testing.T) {
	suite.Run(t, new(NotifierSecurityTestSuite))
}

type NotifierSecurityTestSuite struct {
	suite.Suite
	key string
}

func (s *NotifierSecurityTestSuite) SetupSuite() {
	s.T().Setenv(env.EncNotifierCreds.EnvVar(), "true")
	if !env.EncNotifierCreds.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_ENC_NOTIFIER_CREDS disabled")
		s.T().SkipNow()
	}
	s.key = base64.StdEncoding.EncodeToString([]byte("AES256Key-32Characters1234567890"))
}

func (s *NotifierSecurityTestSuite) TestKeyChainParser() {
	keyChainYaml := `
keyMap:
  0: key1
  1: key2
  2: key3
activeKeyIndex: 2
`
	data := []byte(keyChainYaml)
	expected := &KeyChain{
		KeyMap: map[int]string{
			0: "key1",
			1: "key2",
			2: "key3",
		},
		ActiveKeyIndex: 2,
	}
	keyChain, err := parseKeyChainBytes(data)
	s.Require().NoError(err)
	s.Require().Equal(expected, keyChain)
}

func (s *NotifierSecurityTestSuite) TestGetActiveNotifierEncryptionKey() {
	// case: successful reading keychain
	keyChainFileReader = func(_ string) ([]byte, error) {
		keyChainYaml := `
keyMap:
  0: key1
  1: key2
  2: key3
activeKeyIndex: 2
`
		return []byte(keyChainYaml), nil
	}
	key, idx, err := GetActiveNotifierEncryptionKey()
	s.Require().NoError(err)
	s.Require().Equal("key3", key)
	s.Require().Equal(2, idx)

	// case: error reading file
	keyChainFileReader = func(_ string) ([]byte, error) {
		return nil, errors.New("file not found")
	}
	_, _, err = GetActiveNotifierEncryptionKey()
	s.Require().Error(err)

	// case: active index does not exist
	keyChainFileReader = func(_ string) ([]byte, error) {
		keyChainYaml := `
keyMap:
  0: key1
  1: key2
  2: key3
activeKeyIndex: 100
`
		return []byte(keyChainYaml), nil
	}
	_, _, err = GetActiveNotifierEncryptionKey()
	s.Require().Error(err)
}

func (s *NotifierSecurityTestSuite) TestGetNotifierEncryptionKeyAtIndex() {
	// case: successful reading keychain
	keyChainFileReader = func(_ string) ([]byte, error) {
		keyChainYaml := `
keyMap:
  0: key1
  1: key2
  2: key3
activeKeyIndex: 2
`
		return []byte(keyChainYaml), nil
	}
	key, err := GetNotifierEncryptionKeyAtIndex(1)
	s.Require().NoError(err)
	s.Require().Equal("key2", key)

	// case: index does not exist
	_, err = GetNotifierEncryptionKeyAtIndex(100)
	s.Require().Error(err)

	// case: error reading file
	keyChainFileReader = func(_ string) ([]byte, error) {
		return nil, errors.New("user does not have read permission")
	}
	_, err = GetNotifierEncryptionKeyAtIndex(1)
	s.Require().Error(err)
}

func (s *NotifierSecurityTestSuite) TestSecureCleanupDisabled() {
	s.T().Setenv(env.CleanupNotifierCreds.EnvVar(), "false")
	if env.CleanupNotifierCreds.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_CLEANUP_NOTIFIER_CREDS enabled")
		s.T().SkipNow()
	}

	// Case: secure jira notifier
	jira := &storage.Notifier{
		Type: pkgNotifiers.JiraType,
		Config: &storage.Notifier_Jira{
			Jira: &storage.Jira{
				Password: "fakePassword",
			},
		},
	}
	s.checkUnsecured(jira)
	err := SecureNotifier(jira, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(jira.GetNotifierSecret())
	s.Require().NotEmpty(jira.GetJira().GetPassword())
	s.checkSecured(jira)

	// Case: secure email notifier
	email := &storage.Notifier{
		Type: pkgNotifiers.EmailType,
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				Password: "fakePassword",
			},
		},
	}
	s.checkUnsecured(email)
	err = SecureNotifier(email, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(email.GetNotifierSecret())
	s.Require().NotEmpty(email.GetEmail().GetPassword())
	s.checkSecured(email)

	// Case: secure unauthenticated email notifier
	emailUnauth := &storage.Notifier{
		Type: pkgNotifiers.EmailType,
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				AllowUnauthenticatedSmtp: true,
				Password:                 "",
			},
		},
	}
	s.checkUnsecured(emailUnauth)
	err = SecureNotifier(emailUnauth, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(emailUnauth.GetNotifierSecret())
	s.Require().Empty(emailUnauth.GetEmail().GetPassword())
	s.checkSecured(emailUnauth)

	// Case: secure CSCC notifier
	cscc := &storage.Notifier{
		Type: pkgNotifiers.CSCCType,
		Config: &storage.Notifier_Cscc{
			Cscc: &storage.CSCC{
				ServiceAccount: "fakeServiceAccount",
			},
		},
	}
	s.checkUnsecured(cscc)
	err = SecureNotifier(cscc, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(cscc.GetNotifierSecret())
	s.Require().NotEmpty(cscc.GetCscc().GetServiceAccount())
	s.checkSecured(cscc)

	// Case: secure splunk notifier
	splunk := &storage.Notifier{
		Type: pkgNotifiers.SplunkType,
		Config: &storage.Notifier_Splunk{
			Splunk: &storage.Splunk{
				HttpToken: "fakeHttpToken",
			},
		},
	}
	s.checkUnsecured(splunk)
	err = SecureNotifier(splunk, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(splunk.GetNotifierSecret())
	s.Require().NotEmpty(splunk.GetSplunk().GetHttpToken())
	s.checkSecured(splunk)

	// Case: secure pagerduty notifier
	pagerduty := &storage.Notifier{
		Type: pkgNotifiers.PagerDutyType,
		Config: &storage.Notifier_Pagerduty{
			Pagerduty: &storage.PagerDuty{
				ApiKey: "fakeApiKey",
			},
		},
	}
	s.checkUnsecured(pagerduty)
	err = SecureNotifier(pagerduty, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(pagerduty.GetNotifierSecret())
	s.Require().NotEmpty(pagerduty.GetPagerduty().GetApiKey())
	s.checkSecured(pagerduty)

	// Case: secure generic notifier
	generic := &storage.Notifier{
		Type: pkgNotifiers.GenericType,
		Config: &storage.Notifier_Generic{
			Generic: &storage.Generic{
				Username: "fakeUsername",
				Password: "fakePassword",
			},
		},
	}
	s.checkUnsecured(generic)
	err = SecureNotifier(generic, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(generic.GetNotifierSecret())
	s.Require().NotEmpty(generic.GetGeneric().GetPassword())
	s.checkSecured(generic)

	// Case: secure unauthenticated generic notifier
	genericUnauth := &storage.Notifier{
		Type: pkgNotifiers.GenericType,
		Config: &storage.Notifier_Generic{
			Generic: &storage.Generic{
				Username: "",
				Password: "",
			},
		},
	}
	s.checkUnsecured(genericUnauth)
	err = SecureNotifier(genericUnauth, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(genericUnauth.GetNotifierSecret())
	s.Require().Empty(genericUnauth.GetGeneric().GetPassword())
	s.checkSecured(genericUnauth)

	// Case: secure awsSecurityHub notifier
	awsSecurityHub := &storage.Notifier{
		Type: pkgNotifiers.AWSSecurityHubType,
		Config: &storage.Notifier_AwsSecurityHub{
			AwsSecurityHub: &storage.AWSSecurityHub{
				Credentials: &storage.AWSSecurityHub_Credentials{
					AccessKeyId:     "fakeAccessKeyId",
					SecretAccessKey: "fakeSecretAccessKey",
				},
			},
		},
	}
	s.checkUnsecured(awsSecurityHub)
	err = SecureNotifier(awsSecurityHub, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(awsSecurityHub.GetNotifierSecret())
	s.Require().NotEmpty(awsSecurityHub.GetAwsSecurityHub().GetCredentials().GetAccessKeyId())
	s.Require().NotEmpty(awsSecurityHub.GetAwsSecurityHub().GetCredentials().GetSecretAccessKey())
	s.checkSecured(awsSecurityHub)

	// Case: secure microsoft sentinel notifier
	microsoftSentinel := &storage.Notifier{
		Type: pkgNotifiers.MicrosoftSentinelType,
		Config: &storage.Notifier_MicrosoftSentinel{
			MicrosoftSentinel: &storage.MicrosoftSentinel{
				Secret: "secret value",
			},
		},
	}
	s.checkUnsecured(microsoftSentinel)
	err = SecureNotifier(microsoftSentinel, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(microsoftSentinel.GetNotifierSecret())
	s.Require().NotEmpty(microsoftSentinel.GetMicrosoftSentinel().GetSecret())
	s.checkSecured(microsoftSentinel)

	microsoftSentinel = &storage.Notifier{
		Type: pkgNotifiers.MicrosoftSentinelType,
		Config: &storage.Notifier_MicrosoftSentinel{
			MicrosoftSentinel: &storage.MicrosoftSentinel{
				ClientCertAuthConfig: &storage.MicrosoftSentinel_ClientCertAuthConfig{
					PrivateKey: "private key",
					ClientCert: "client cert",
				},
			},
		},
	}
	s.checkUnsecured(microsoftSentinel)
	err = SecureNotifier(microsoftSentinel, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(microsoftSentinel.GetNotifierSecret())
	s.Require().NotEmpty(microsoftSentinel.GetMicrosoftSentinel().GetClientCertAuthConfig().GetPrivateKey())
	s.Require().NotEmpty(microsoftSentinel.GetMicrosoftSentinel().GetClientCertAuthConfig().GetClientCert())
	s.checkSecured(microsoftSentinel)
}

func (s *NotifierSecurityTestSuite) TestSecureCleanupEnabled() {
	s.T().Setenv(env.CleanupNotifierCreds.EnvVar(), "true")
	if !env.CleanupNotifierCreds.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_CLEANUP_NOTIFIER_CREDS disabled")
		s.T().SkipNow()
	}
	// Case: secure jira notifier
	jira := &storage.Notifier{
		Type: pkgNotifiers.JiraType,
		Config: &storage.Notifier_Jira{
			Jira: &storage.Jira{
				Password: "fakePassword",
			},
		},
	}
	s.checkUnsecured(jira)
	err := SecureNotifier(jira, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(jira.GetNotifierSecret())
	s.Require().Empty(jira.GetJira().GetPassword())
	s.checkSecured(jira)

	// Case: secure email notifier
	email := &storage.Notifier{
		Type: pkgNotifiers.EmailType,
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				Password: "fakePassword",
			},
		},
	}
	s.checkUnsecured(email)
	err = SecureNotifier(email, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(email.GetNotifierSecret())
	s.Require().Empty(email.GetEmail().GetPassword())
	s.checkSecured(email)

	// Case: secure unauthenticated email notifier
	emailUnauth := &storage.Notifier{
		Type: pkgNotifiers.EmailType,
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				AllowUnauthenticatedSmtp: true,
				Password:                 "",
			},
		},
	}
	s.checkUnsecured(emailUnauth)
	err = SecureNotifier(emailUnauth, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(emailUnauth.GetNotifierSecret())
	s.Require().Empty(emailUnauth.GetEmail().GetPassword())
	s.checkSecured(emailUnauth)

	// Case: secure CSCC notifier
	cscc := &storage.Notifier{
		Type: pkgNotifiers.CSCCType,
		Config: &storage.Notifier_Cscc{
			Cscc: &storage.CSCC{
				ServiceAccount: "fakeServiceAccount",
			},
		},
	}
	s.checkUnsecured(cscc)
	err = SecureNotifier(cscc, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(cscc.GetNotifierSecret())
	s.Require().Empty(cscc.GetCscc().GetServiceAccount())
	s.checkSecured(cscc)

	// Case: secure splunk notifier
	splunk := &storage.Notifier{
		Type: pkgNotifiers.SplunkType,
		Config: &storage.Notifier_Splunk{
			Splunk: &storage.Splunk{
				HttpToken: "fakeHttpToken",
			},
		},
	}
	s.checkUnsecured(splunk)
	err = SecureNotifier(splunk, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(splunk.GetNotifierSecret())
	s.Require().Empty(splunk.GetSplunk().GetHttpToken())
	s.checkSecured(splunk)

	// Case: secure pagerduty notifier
	pagerduty := &storage.Notifier{
		Type: pkgNotifiers.PagerDutyType,
		Config: &storage.Notifier_Pagerduty{
			Pagerduty: &storage.PagerDuty{
				ApiKey: "fakeApiKey",
			},
		},
	}
	s.checkUnsecured(pagerduty)
	err = SecureNotifier(pagerduty, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(pagerduty.GetNotifierSecret())
	s.Require().Empty(pagerduty.GetPagerduty().GetApiKey())
	s.checkSecured(pagerduty)

	// Case: secure generic notifier
	generic := &storage.Notifier{
		Type: pkgNotifiers.GenericType,
		Config: &storage.Notifier_Generic{
			Generic: &storage.Generic{
				Username: "fakeUsername",
				Password: "fakePassword",
			},
		},
	}
	s.checkUnsecured(generic)
	err = SecureNotifier(generic, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(generic.GetNotifierSecret())
	s.Require().Empty(generic.GetGeneric().GetPassword())
	s.checkSecured(generic)

	// Case: secure unauthenticated generic notifier
	genericUnauth := &storage.Notifier{
		Type: pkgNotifiers.GenericType,
		Config: &storage.Notifier_Generic{
			Generic: &storage.Generic{
				Username: "",
				Password: "",
			},
		},
	}
	s.checkUnsecured(genericUnauth)
	err = SecureNotifier(genericUnauth, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(genericUnauth.GetNotifierSecret())
	s.Require().Empty(genericUnauth.GetGeneric().GetPassword())
	s.checkSecured(genericUnauth)

	// Case: secure awsSecurityHub notifier
	awsSecurityHub := &storage.Notifier{
		Type: pkgNotifiers.AWSSecurityHubType,
		Config: &storage.Notifier_AwsSecurityHub{
			AwsSecurityHub: &storage.AWSSecurityHub{
				Credentials: &storage.AWSSecurityHub_Credentials{
					AccessKeyId:     "fakeAccessKeyId",
					SecretAccessKey: "fakeSecretAccessKey",
				},
			},
		},
	}
	s.checkUnsecured(awsSecurityHub)
	err = SecureNotifier(awsSecurityHub, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(awsSecurityHub.GetNotifierSecret())
	s.Require().Empty(awsSecurityHub.GetAwsSecurityHub().GetCredentials().GetAccessKeyId())
	s.Require().Empty(awsSecurityHub.GetAwsSecurityHub().GetCredentials().GetSecretAccessKey())
	s.checkSecured(awsSecurityHub)

	// Case: secure microsoft sentinel notifier
	microsoftSentinel := &storage.Notifier{
		Type: pkgNotifiers.MicrosoftSentinelType,
		Config: &storage.Notifier_MicrosoftSentinel{
			MicrosoftSentinel: &storage.MicrosoftSentinel{
				Secret: "secret value",
			},
		},
	}
	s.checkUnsecured(microsoftSentinel)
	err = SecureNotifier(microsoftSentinel, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(microsoftSentinel.GetNotifierSecret())
	s.Require().Empty(microsoftSentinel.GetMicrosoftSentinel().GetSecret())
	s.Require().Empty(microsoftSentinel.GetMicrosoftSentinel().GetClientCertAuthConfig().GetPrivateKey())
	s.checkSecured(microsoftSentinel)

	microsoftSentinel.GetMicrosoftSentinel().ClientCertAuthConfig = &storage.MicrosoftSentinel_ClientCertAuthConfig{
		PrivateKey: "private key",
		ClientCert: "client cert",
	}
	s.checkUnsecured(microsoftSentinel)
	err = SecureNotifier(microsoftSentinel, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(microsoftSentinel.GetNotifierSecret())
	s.Require().Empty(microsoftSentinel.GetMicrosoftSentinel().GetSecret())
	s.Require().Empty(microsoftSentinel.GetMicrosoftSentinel().GetClientCertAuthConfig().GetPrivateKey())
	s.Require().NotEmpty(microsoftSentinel.GetMicrosoftSentinel().GetClientCertAuthConfig().GetClientCert())
	s.checkSecured(microsoftSentinel)
}

func (s *NotifierSecurityTestSuite) checkSecured(notifier *storage.Notifier) {
	secured, err := IsNotifierSecured(notifier)
	s.Require().NoError(err)
	s.Require().True(secured)
}

func (s *NotifierSecurityTestSuite) checkUnsecured(notifier *storage.Notifier) {
	secured, err := IsNotifierSecured(notifier)
	s.Require().NoError(err)
	s.Require().False(secured)
}

func (s *NotifierSecurityTestSuite) TestRekeyNotifier() {
	notifier := &storage.Notifier{
		Type: pkgNotifiers.EmailType,
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				Password: "fakePassword",
			},
		},
	}
	err := SecureNotifier(notifier, s.key)
	s.Require().NoError(err)
	s.Require().NotEmpty(notifier.GetNotifierSecret())
	oldSecret := notifier.GetNotifierSecret()
	newKey := base64.StdEncoding.EncodeToString([]byte("New256Key-32Characters1234567890"))
	err = RekeyNotifier(notifier, s.key, newKey)
	s.Require().NoError(err)
	s.Require().NotEmpty(notifier.GetNotifierSecret())
	s.Require().NotEqual(oldSecret, notifier.GetNotifierSecret())
}
