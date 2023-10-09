package utils

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/env"
	pkgNotifiers "github.com/stackrox/rox/pkg/notifiers"
)

const (
	encryptionKeyFile = "/run/secrets/stackrox.io/central-encryption-key/encryption-key"
)

// GetNotifierSecretEncryptionKey returns the key for encrypting/decrypting notifier secrets
func GetNotifierSecretEncryptionKey() (string, error) {
	key, err := os.ReadFile(encryptionKeyFile)
	if err != nil {
		return "", errors.Wrap(err, "Could not load notifier encryption key")
	}
	return string(key), nil
}

// SecureNotifier secures the secrets in the given notifier
func SecureNotifier(notifier *storage.Notifier, cryptoCodec cryptoutils.CryptoCodec, key string) error {
	if !env.EncNotifierCreds.BooleanSetting() {
		return nil
	}
	if notifier.GetConfig() == nil {
		return nil
	}
	secret := ""
	var err error
	switch notifier.GetType() {
	case pkgNotifiers.JiraType:
		jira := notifier.GetJira()
		if jira == nil {
			return nil
		}
		secret, err = cryptoCodec.Encrypt(key, jira.GetPassword())
		if err != nil {
			return err
		}
	case pkgNotifiers.EmailType:
		email := notifier.GetEmail()
		if email == nil {
			return nil
		}
		secret, err = cryptoCodec.Encrypt(key, email.GetPassword())
		if err != nil {
			return err
		}
	case pkgNotifiers.CSCCType:
		cscc := notifier.GetCscc()
		if cscc == nil {
			return nil
		}
		secret, err = cryptoCodec.Encrypt(key, cscc.GetServiceAccount())
		if err != nil {
			return err
		}
	case pkgNotifiers.SplunkType:
		splunk := notifier.GetSplunk()
		if splunk == nil {
			return nil
		}
		secret, err = cryptoCodec.Encrypt(key, splunk.GetHttpToken())
		if err != nil {
			return err
		}
	case pkgNotifiers.PagerDutyType:
		pagerDuty := notifier.GetPagerduty()
		if pagerDuty == nil {
			return nil
		}
		secret, err = cryptoCodec.Encrypt(key, pagerDuty.GetApiKey())
		if err != nil {
			return err
		}
	case pkgNotifiers.GenericType:
		generic := notifier.GetGeneric()
		if generic == nil {
			return nil
		}
		secret, err = cryptoCodec.Encrypt(key, generic.GetPassword())
		if err != nil {
			return err
		}
	case pkgNotifiers.AWSSecurityHubType:
		awsSecurityHub := notifier.GetAwsSecurityHub()
		if awsSecurityHub == nil {
			return nil
		}
		creds := awsSecurityHub.GetCredentials()
		if creds == nil {
			return nil
		}
		marshalled, err := creds.Marshal()
		if err != nil {
			return err
		}
		secret, err = cryptoCodec.Encrypt(key, string(marshalled))
		if err != nil {
			return err
		}

	}
	notifier.NotifierSecret = secret
	cleanupCreds(notifier)
	return nil
}

func cleanupCreds(notifier *storage.Notifier) {
	if !env.CleanupNotifierCreds.BooleanSetting() {
		return
	}
	switch notifier.GetType() {
	case pkgNotifiers.JiraType:
		jira := notifier.GetJira()
		if jira == nil {
			return
		}
		jira.Password = ""
	case pkgNotifiers.EmailType:
		email := notifier.GetEmail()
		if email == nil {
			return
		}
		email.Password = ""
	case pkgNotifiers.CSCCType:
		cscc := notifier.GetCscc()
		if cscc == nil {
			return
		}
		cscc.ServiceAccount = ""
	case pkgNotifiers.SplunkType:
		splunk := notifier.GetSplunk()
		if splunk == nil {
			return
		}
		splunk.HttpToken = ""
	case pkgNotifiers.PagerDutyType:
		pagerDuty := notifier.GetPagerduty()
		if pagerDuty == nil {
			return
		}
		pagerDuty.ApiKey = ""
	case pkgNotifiers.GenericType:
		generic := notifier.GetGeneric()
		if generic == nil {
			return
		}
		generic.Password = ""
	case pkgNotifiers.AWSSecurityHubType:
		awsSecurityHub := notifier.GetAwsSecurityHub()
		if awsSecurityHub == nil {
			return
		}
		awsSecurityHub.Credentials = nil
	}
}
