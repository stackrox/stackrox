package notifiers

// EncryptedNotifier is the notifier that encrypts its credentials
//
//go:generate mockgen-wrapper EncryptedNotifier
type EncryptedNotifier interface {
	Notifier
	EncryptCredentials(key string) error
}
