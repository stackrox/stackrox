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

// SecureNotifier secures the secrets in the given notifier and returns true if the encrypted creds were modified, false otherwise	
func SecureNotifier(notifier *storage.Notifier, key string) (bool, error) {
	if !env.EncNotifierCreds.BooleanSetting() {
		return false, nil
	}
	if notifier.GetConfig() == nil {
		return false, nil
	}
	encCreds := notifier.GetNotifierSecret()

	cryptoCodec := cryptocodec.Singleton()
	var err error
	switch notifier.GetType() {
	case pkgNotifiers.JiraType:
		jira := notifier.GetJira()
		if jira == nil {
			return false, nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, jira.GetPassword())
		return notifier.NotifierSecret != encCreds, err

	case pkgNotifiers.EmailType:
		email := notifier.GetEmail()
		if email == nil {
			return false, nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, email.GetPassword())
		return notifier.NotifierSecret != encCreds, err

	case pkgNotifiers.CSCCType:
		cscc := notifier.GetCscc()
		if cscc == nil {
			return false, nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, cscc.GetServiceAccount())
		return notifier.NotifierSecret != encCreds, err

	case pkgNotifiers.SplunkType:
		splunk := notifier.GetSplunk()
		if splunk == nil {
			return false, nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, splunk.GetHttpToken())
		return notifier.NotifierSecret != encCreds, err

	case pkgNotifiers.PagerDutyType:
		pagerDuty := notifier.GetPagerduty()
		if pagerDuty == nil {
			return false, nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, pagerDuty.GetApiKey())
		return notifier.NotifierSecret != encCreds, err

	case pkgNotifiers.GenericType:
		generic := notifier.GetGeneric()
		if generic == nil {
			return false, nil
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, generic.GetPassword())
		return notifier.NotifierSecret != encCreds, err

	case pkgNotifiers.AWSSecurityHubType:
		awsSecurityHub := notifier.GetAwsSecurityHub()
		if awsSecurityHub == nil {
			return false, nil
		}
		creds := awsSecurityHub.GetCredentials()
		if creds == nil {
			return false, nil
		}
		marshalled, err := creds.Marshal()
		if err != nil {
			return false, err
		}
		notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, string(marshalled))
		return notifier.NotifierSecret != encCreds, err
	}
	// TODO (ROX-19879): Cleanup creds if ROX_CLEANUP_NOTIFIER_CREDS is enabled
	return false, nil
}
