package aws

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/secretinformer"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	cloudCredentialsKey = "credentials"
)

type awsCredentialsManagerImpl struct {
	namespace        string
	secretName       string
	informer         *secretinformer.SecretInformer
	secretFound      bool
	mirroredFileName string
	mutex            sync.RWMutex
}

var _ CredentialsManager = &awsCredentialsManagerImpl{}

// NewCredentialsManager creates a new AWS credential manager.
func NewCredentialsManager(
	k8sClient kubernetes.Interface,
	namespace string,
	secretName string,
	mirroredFileName string,
) CredentialsManager {
	return newAWSCredentialsManagerImpl(k8sClient, namespace, secretName, mirroredFileName)
}

func newAWSCredentialsManagerImpl(
	k8sClient kubernetes.Interface,
	namespace string,
	secretName string,
	mirroredFileName string,
) *awsCredentialsManagerImpl {
	mgr := &awsCredentialsManagerImpl{
		namespace:        namespace,
		secretName:       secretName,
		mirroredFileName: mirroredFileName,
	}
	mgr.informer = secretinformer.NewSecretInformer(
		namespace,
		secretName,
		k8sClient,
		mgr.updateSecret,
		mgr.updateSecret,
		mgr.deleteSecret,
	)
	return mgr
}

func (c *awsCredentialsManagerImpl) updateSecret(secret *v1.Secret) {
	if stsConfig, ok := secret.Data[cloudCredentialsKey]; ok {
		if len(stsConfig) == 0 {
			c.deleteSecret()
			return
		}

		c.mutex.Lock()
		defer c.mutex.Unlock()
		if err := mirrorToLocalFile([]byte(stsConfig), c.mirroredFileName); err != nil {
			log.Errorf(
				"Failed to mirror AWS cloud credential file for %q for %s/%s: ",
				c.mirroredFileName,
				c.namespace,
				c.secretName,
				err,
			)
			c.secretFound = false
			return
		}
		c.secretFound = true
		log.Infof("Updated AWS cloud credentials based on %s/%s", c.namespace, c.secretName)
	}
}

func (c *awsCredentialsManagerImpl) deleteSecret() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.secretFound = false
	if err := os.Remove(c.mirroredFileName); err != nil {
		log.Errorf("Could not remove mirrored credentials file %q", c.mirroredFileName)
	}
	log.Infof("Deleted AWS cloud credentials based on %s/%s", c.namespace, c.secretName)
}

func (c *awsCredentialsManagerImpl) Start() {
	if err := c.informer.Start(); err != nil {
		log.Error("Failed to start AWS cloud credentials manager: ", err)
	}
}

func (c *awsCredentialsManagerImpl) Stop() {
	c.informer.Stop()
	if err := os.Remove(c.mirroredFileName); err != nil {
		log.Errorf(
			"Could not remove mirrored credentials file %q for %s/%s: ",
			c.mirroredFileName,
			c.namespace,
			c.secretName,
			err,
		)
	}
}

// NewSession returns an AWS session based on the environment.
//
// The following sources are considered:
//  1. Cloud credentials secret (stackrox/aws-cloud-credentials) containing the STS configuration
//     for pod IAM roles. Ignored if the secret does not exist.
//  2. The default AWS credentials chain based on the pod's environment and metadata.
func (c *awsCredentialsManagerImpl) NewSession(cfgs ...*aws.Config) (*session.Session, error) {
	opts := session.Options{}
	opts.Config.MergeIn(cfgs...)

	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.secretFound {
		opts.SharedConfigState = session.SharedConfigEnable
		opts.SharedConfigFiles = []string{c.mirroredFileName}
	}

	return session.NewSessionWithOptions(opts)
}

func mirrorToLocalFile(data []byte, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to create AWS cloud credentials file %q", filename)
	}
	defer utils.IgnoreError(file.Close)

	if _, err := file.Write(data); err != nil {
		return errors.Wrapf(err, "failed to write AWS cloud credentials to %q", filename)
	}
	return nil
}
