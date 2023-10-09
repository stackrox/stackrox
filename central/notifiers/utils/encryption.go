package utils

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
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
func SecureNotifier(notifier *storage.Notifier, key string) error {
	if !env.EncNotifierCreds.BooleanSetting() {
		return nil
	}
	if notifier.GetConfig() == nil {
		return nil
	}
	cryptoCodec := cryptocodec.Singleton()
	var err error
	switch notifier.GetType() {
	case pkgNotifiers.JiraType:
		jira := notifier.GetJira()
		if jira == nil {
			return nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, jira.GetPassword())
		if err != nil {
			return err
		}
	case pkgNotifiers.EmailType:
		email := notifier.GetEmail()
		if email == nil {
			return nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, email.GetPassword())
		if err != nil {
			return err
		}
	case pkgNotifiers.CSCCType:
		cscc := notifier.GetCscc()
		if cscc == nil {
			return nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, cscc.GetServiceAccount())
		if err != nil {
			return err
		}
	case pkgNotifiers.SplunkType:
		splunk := notifier.GetSplunk()
		if splunk == nil {
			return nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, splunk.GetHttpToken())
		if err != nil {
			return err
		}
	case pkgNotifiers.PagerDutyType:
		pagerDuty := notifier.GetPagerduty()
		if pagerDuty == nil {
			return nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, pagerDuty.GetApiKey())
		if err != nil {
			return err
		}
	case pkgNotifiers.GenericType:
		generic := notifier.GetGeneric()
		if generic == nil {
			return nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, generic.GetPassword())
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
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, string(marshalled))
		if err != nil {
			return err
		}
	}
	// TODO (ROX-19879): Cleanup creds if ROX_CLEANUP_NOTIFIER_CREDS is enabled
	return nil
}
