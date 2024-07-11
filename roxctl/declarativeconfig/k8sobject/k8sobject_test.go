package k8sobject

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestReadFromK8sObject_ConfigMap(t *testing.T) {
	cases := map[string]struct {
		cm        *corev1.ConfigMap
		configMap string
		namespace string
		fail      bool
		contents  [][]byte
	}{
		"reading from non-existing config map should return an error": {
			configMap: "test-cm",
			namespace: "testing",
			fail:      true,
		},
		"reading from an existing config map with empty data should not return an error": {
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
					Namespace: "testing",
				},
			},
			configMap: "test-cm",
			namespace: "testing",
			contents:  [][]byte{},
		},
		"reading from an existing config map with data should return the data": {
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
					Namespace: "testing",
				},
				Data: map[string]string{
					"some-key": "some-stuff",
				},
			},
			configMap: "test-cm",
			namespace: "testing",
			contents:  [][]byte{[]byte("some-stuff")},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			if c.cm != nil {
				client = fake.NewSimpleClientset(c.cm)
			}
			contents, err := readConfigMap(context.Background(), client, c.configMap, c.namespace)
			if c.fail {
				assert.Error(t, err)
				assert.Nil(t, contents)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.contents, contents)
			}
		})
	}
}

func TestWriteToK8sObject_ConfigMap(t *testing.T) {
	cases := map[string]struct {
		cm        *corev1.ConfigMap
		configMap string
		namespace string
		key       string
		write     []byte
		fail      bool
		contents  [][]byte
	}{
		"writing to non-existing config map should fail": {
			configMap: "test-cm",
			key:       "test-key",
			namespace: "testing",
			fail:      true,
		},
		"writing to empty config map shouldn't fail": {
			configMap: "test-cm",
			key:       "test-key",
			namespace: "testing",
			write:     []byte("test-stuff"),
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
					Namespace: "testing",
				},
			},
			contents: [][]byte{[]byte("test-stuff")},
		},
		"writing to non-empty config map shouldn't fail": {
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
					Namespace: "testing",
				},
				Data: map[string]string{
					"some-key": "some-stuff",
				},
			},
			configMap: "test-cm",
			namespace: "testing",
			write:     []byte("test-stuff"),
			key:       "test-key",
			contents:  [][]byte{[]byte("test-stuff"), []byte("some-stuff")},
		},
		"overriding key in config map shouldn't fail": {
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
					Namespace: "testing",
				},
				Data: map[string]string{
					"test-key": "some-stuff",
				},
			},
			configMap: "test-cm",
			namespace: "testing",
			key:       "test-key",
			write:     []byte("test-stuff"),
			contents:  [][]byte{[]byte("test-stuff")},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			if c.cm != nil {
				client = fake.NewSimpleClientset(c.cm)
			}
			err := writeConfigMap(context.Background(), client, c.configMap, c.namespace, c.key, c.write)
			if c.fail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				contents, err := readConfigMap(context.Background(), client, c.configMap, c.namespace)
				assert.NoError(t, err)
				assert.ElementsMatch(t, c.contents, contents)
			}
		})
	}
}

func TestReadFromK8sObject_Secret(t *testing.T) {
	cases := map[string]struct {
		sec       *corev1.Secret
		secret    string
		namespace string
		fail      bool
		contents  [][]byte
	}{
		"reading from non-existent secret should return an error": {
			secret:    "test-secret",
			namespace: "testing",
			fail:      true,
		},
		"reading from empty secret shouldn't return an error": {
			sec: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "testing",
				},
			},
			secret:    "test-secret",
			namespace: "testing",
			contents:  [][]byte{},
		},
		"reading from non-empty secret shouldn't return an error and expected contents": {
			sec: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					"some-key": []byte("testing"),
				},
			},
			secret:    "test-secret",
			namespace: "testing",
			contents:  [][]byte{[]byte("testing")},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			if c.sec != nil {
				client = fake.NewSimpleClientset(c.sec)
			}
			contents, err := readSecret(context.Background(), client, c.secret, c.namespace)
			if c.fail {
				assert.Error(t, err)
				assert.Nil(t, contents)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.contents, contents)
			}
		})
	}
}

func TestWriteToK8sObject_Secret(t *testing.T) {
	cases := map[string]struct {
		sec       *corev1.Secret
		secret    string
		namespace string
		key       string
		write     []byte
		fail      bool
		contents  [][]byte
	}{
		"writing to non-existing secret should fail": {
			secret:    "test-secret",
			key:       "test-key",
			namespace: "testing",
			fail:      true,
		},
		"writing to empty secret shouldn't fail": {
			secret:    "test-secret",
			key:       "test-key",
			namespace: "testing",
			write:     []byte("test-stuff"),
			sec: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "testing",
				},
			},
			contents: [][]byte{[]byte("test-stuff")},
		},
		"writing to non-empty secret shouldn't fail": {
			sec: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					"some-key": []byte("testing"),
				},
			},
			secret:    "test-secret",
			namespace: "testing",
			write:     []byte("test-stuff"),
			key:       "test-key",
			contents:  [][]byte{[]byte("test-stuff"), []byte("testing")},
		},
		"overriding key in secret shouldn't fail": {
			sec: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					"test-key": []byte("testing"),
				},
			},
			secret:    "test-secret",
			namespace: "testing",
			key:       "test-key",
			write:     []byte("test-stuff"),
			contents:  [][]byte{[]byte("test-stuff")},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			if c.sec != nil {
				client = fake.NewSimpleClientset(c.sec)
			}
			err := writeSecret(context.Background(), client, c.secret, c.namespace, c.key, c.write)
			if c.fail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				contents, err := readSecret(context.Background(), client, c.secret, c.namespace)
				assert.NoError(t, err)
				assert.ElementsMatch(t, c.contents, contents)
			}
		})
	}
}
