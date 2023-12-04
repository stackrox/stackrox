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

// SecureNotifier secures the secrets in the given notifier and returns true if the encrypted creds were modified,
// false otherwise
func SecureNotifier(notifier *storage.Notifier, key string) (bool, error) {
	if !env.EncNotifierCreds.BooleanSetting() {
		return false, nil
	}
	if notifier.GetConfig() == nil {
		return false, nil
	}

	cryptoCodec := cryptocodec.Singleton()
	var err error
	switch notifier.GetType() {
	case pkgNotifiers.JiraType:
		jira := notifier.GetJira()
		if jira != nil && jira.GetPassword() != "" {
			notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, jira.GetPassword())
			if err != nil {
				return false, err
			}
			jira.Password = ""
			return true, nil
		}

	case pkgNotifiers.EmailType:
		email := notifier.GetEmail()
		if email != nil && email.GetPassword() != "" {
			notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, email.GetPassword())
			if err != nil {
				return false, err
			}
			email.Password = ""
			return true, nil
		}

	case pkgNotifiers.CSCCType:
		cscc := notifier.GetCscc()
		if cscc != nil && cscc.GetServiceAccount() != "" {
			notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, cscc.GetServiceAccount())
			if err != nil {
				return false, err
			}
			cscc.ServiceAccount = ""
			return true, nil
		}

	case pkgNotifiers.SplunkType:
		splunk := notifier.GetSplunk()
		if splunk != nil && splunk.GetHttpToken() != "" {
			notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, splunk.GetHttpToken())
			if err != nil {
				return false, err
			}
			splunk.HttpToken = ""
			return true, nil
		}

	case pkgNotifiers.PagerDutyType:
		pagerDuty := notifier.GetPagerduty()
		if pagerDuty != nil && pagerDuty.GetApiKey() != "" {
			notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, pagerDuty.GetApiKey())
			if err != nil {
				return false, err
			}
			pagerDuty.ApiKey = ""
			return true, nil
		}

	case pkgNotifiers.GenericType:
		generic := notifier.GetGeneric()
		if generic != nil && generic.GetPassword() != "" {
			notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, generic.GetPassword())
			if err != nil {
				return false, err
			}
			generic.Password = ""
			return true, nil
		}

	case pkgNotifiers.AWSSecurityHubType:
		awsSecurityHub := notifier.GetAwsSecurityHub()
		if awsSecurityHub == nil {
			return false, nil
		}
		creds := awsSecurityHub.GetCredentials()
		if creds != nil && creds.GetAccessKeyId() != "" && creds.GetSecretAccessKey() != "" {
			marshalled, err := creds.Marshal()
			if err != nil {
				return false, err
			}
			notifier.NotifierSecret, err = cryptoCodec.Encrypt(key, string(marshalled))
			if err != nil {
				return false, err
			}
			creds.AccessKeyId = ""
			creds.SecretAccessKey = ""
			return true, nil
		}
	}
	return false, nil
}
