package utils

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
	pkgNotifiers "github.com/stackrox/rox/pkg/notifiers"
	"gopkg.in/yaml.v3"
)

const (
	encryptionKeyChainFile = "/run/secrets/stackrox.io/central-encryption-key-chain/key-chain.yaml"
)

// KeyChain contains the keychain for notifier crypto
type KeyChain struct {
	KeyMap         map[int]string `yaml:"keyMap"`
	ActiveKeyIndex int            `yaml:"activeKeyIndex"`
}

var keyChainFileReader = os.ReadFile

// GetActiveNotifierEncryptionKey returns the active key for encrypting/decrypting notifier secrets and the index of
// the active key in the keychain
func GetActiveNotifierEncryptionKey() (string, int, error) {
	data, err := keyChainFileReader(encryptionKeyChainFile)
	if err != nil {
		return "", 0, errors.Wrap(err, "Could not load notifier encryption keychain")
	}
	keyChain, err := parseKeyChainBytes(data)
	if err != nil {
		return "", 0, err
	}
	key, exists := keyChain.KeyMap[keyChain.ActiveKeyIndex]
	if !exists {
		return "", 0, errors.New("Invalid keychain. Encryption key at active index does not exist")
	}
	return key, keyChain.ActiveKeyIndex, nil
}

// GetNotifierEncryptionKeyAtIndex returns the key at the given index from the keychain
func GetNotifierEncryptionKeyAtIndex(idx int) (string, error) {
	data, err := keyChainFileReader(encryptionKeyChainFile)
	if err != nil {
		return "", errors.Wrap(err, "Could not load notifier encryption keychain")
	}
	keyChain, err := parseKeyChainBytes(data)
	if err != nil {
		return "", err
	}
	key, exists := keyChain.KeyMap[idx]
	if !exists {
		return "", errors.Errorf("Encryption key index '%d' does not exist", idx)
	}
	return key, nil
}

func parseKeyChainBytes(data []byte) (*KeyChain, error) {
	var chain KeyChain
	err := yaml.Unmarshal(data, &chain)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing notifier encryption keychain")
	}
	return &chain, nil
}

// SecureNotifier secures the secrets in the given unsecured notifier
func SecureNotifier(notifier *storage.Notifier, key string) error {
	if !env.EncNotifierCreds.BooleanSetting() {
		return nil
	}
	if notifier.GetConfig() == nil {
		return nil
	}
	secured, err := IsNotifierSecured(notifier)
	if err != nil {
		return err
	}
	if secured {
		return nil
	}
	creds, err := getCredentials(notifier)
	if err != nil {
		return err
	}

	cryptoCodec := cryptocodec.Singleton()
	notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, creds)
	if err != nil {
		return err
	}
	if env.CleanupNotifierCreds.BooleanSetting() {
		cleanupCredentials(notifier)
	}
	return nil
}

// IsNotifierSecured returns true if the given notifier is already secured
func IsNotifierSecured(notifier *storage.Notifier) (bool, error) {
	if !env.EncNotifierCreds.BooleanSetting() {
		return false, nil
	}
	if !env.CleanupNotifierCreds.BooleanSetting() {
		// If cleanup is disabled, creds do not have to be cleaned when a notifier is secured.
		// So just checking if the field NotifierSecret is non-empty is enough.
		return notifier.GetNotifierSecret() != "", nil
	}
	creds, err := getCredentials(notifier)
	if err != nil {
		return false, nil
	}
	if notifier.GetType() == pkgNotifiers.AWSSecurityHubType {
		creds := notifier.GetAwsSecurityHub().GetCredentials()
		return notifier.GetNotifierSecret() != "" && creds.GetAccessKeyId() == "" && creds.GetSecretAccessKey() == "", nil
	}
	return notifier.GetNotifierSecret() != "" && creds == "", nil
}

// RekeyNotifier rekeys an already secured notifier using the new key
func RekeyNotifier(notifier *storage.Notifier, oldKey string, newKey string) error {
	if !env.EncNotifierCreds.BooleanSetting() {
		return nil
	}
	secured, err := IsNotifierSecured(notifier)
	if err != nil {
		return err
	}
	if !secured {
		return errors.New("Cannot rekey unsecured notifier")
	}
	cryptoCodec := cryptocodec.Singleton()
	creds, err := cryptoCodec.Decrypt(oldKey, notifier.GetNotifierSecret())
	if err != nil {
		return err
	}
	notifier.NotifierSecret, err = cryptoCodec.Encrypt(newKey, creds)
	return err
}

func getCredentials(notifier *storage.Notifier) (string, error) {
	if notifier.GetConfig() == nil {
		return "", nil
	}
	switch notifier.GetType() {
	case pkgNotifiers.JiraType:
		return notifier.GetJira().GetPassword(), nil
	case pkgNotifiers.EmailType:
		return notifier.GetEmail().GetPassword(), nil
	case pkgNotifiers.CSCCType:
		return notifier.GetCscc().GetServiceAccount(), nil
	case pkgNotifiers.SplunkType:
		return notifier.GetSplunk().GetHttpToken(), nil
	case pkgNotifiers.PagerDutyType:
		return notifier.GetPagerduty().GetApiKey(), nil
	case pkgNotifiers.GenericType:
		return notifier.GetGeneric().GetPassword(), nil
	case pkgNotifiers.AWSSecurityHubType:
		creds := notifier.GetAwsSecurityHub().GetCredentials()
		if creds != nil {
			marshalled, err := creds.MarshalVT()
			if err != nil {
				return "", err
			}
			return string(marshalled), nil
		}
	}
	return "", nil
}

func cleanupCredentials(notifier *storage.Notifier) {
	if notifier.GetConfig() == nil {
		return
	}
	switch notifier.GetType() {
	case pkgNotifiers.JiraType:
		jira := notifier.GetJira()
		if jira != nil {
			jira.Password = ""
		}
	case pkgNotifiers.EmailType:
		email := notifier.GetEmail()
		if email != nil {
			email.Password = ""
		}
	case pkgNotifiers.CSCCType:
		cscc := notifier.GetCscc()
		if cscc != nil {
			cscc.ServiceAccount = ""
		}
	case pkgNotifiers.SplunkType:
		splunk := notifier.GetSplunk()
		if splunk != nil {
			splunk.HttpToken = ""
		}
	case pkgNotifiers.PagerDutyType:
		pagerDuty := notifier.GetPagerduty()
		if pagerDuty != nil {
			pagerDuty.ApiKey = ""
		}
	case pkgNotifiers.GenericType:
		generic := notifier.GetGeneric()
		if generic != nil {
			generic.Password = ""
		}
	case pkgNotifiers.AWSSecurityHubType:
		creds := notifier.GetAwsSecurityHub().GetCredentials()
		if creds != nil {
			creds.AccessKeyId = ""
			creds.SecretAccessKey = ""
		}
	}
}
