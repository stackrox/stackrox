package securedcluster

import "fmt"

// InitBundleSecretsMissingError returns an error message for when init-bundle secrets are missing.
func InitBundleSecretsMissingError(namespace string) error {
	return fmt.Errorf("some init-bundle secrets missing in namespace %q, "+
		"please make sure you have downloaded init-bundle secrets (from UI or with roxctl) "+
		"and created corresponding resources in the correct namespace", namespace)
}
