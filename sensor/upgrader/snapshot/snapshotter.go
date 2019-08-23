package snapshot

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var (
	log = logging.LoggerForModule()

	jsonSeparator = []byte("\x00")
)

type snapshotter struct {
	ctx *upgradectx.UpgradeContext
}

func (s *snapshotter) SnapshotState() ([]k8sobjects.Object, error) {
	coreV1Client := s.ctx.ClientSet().CoreV1()

	snapshotSecret, err := coreV1Client.Secrets(common.Namespace).Get(secretName, metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		snapshotSecret = nil
		err = nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "retrieving state snapshot secret")
	}

	if snapshotSecret != nil {
		log.Info("Matching state snapshot secret found, not creating a new one")
		return s.stateFromSecret(snapshotSecret)
	}

	objects, snapshotSecret, err := s.createStateSnapshot()
	if err != nil {
		return nil, errors.Wrap(err, "snapshotting state")
	}
	_, err = coreV1Client.Secrets(common.Namespace).Create(snapshotSecret)
	if err != nil {
		return nil, errors.Wrap(err, "creating state snapshot secret")
	}
	return objects, nil
}

func (s *snapshotter) stateFromSecret(secret *v1.Secret) ([]k8sobjects.Object, error) {
	if processID := secret.Labels[common.UpgradeProcessIDLabelKey]; processID != s.ctx.ProcessID() {
		return nil, errors.Errorf("state snapshot secret belongs to wrong upgrade process %q, expected %s", processID, s.ctx.ProcessID())
	}

	gzData := secret.Data[secretDataName]
	if len(gzData) == 0 {
		return nil, errors.New("state snapshot secret contains no relevant data")
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(gzData))
	if err != nil {
		return nil, errors.Wrap(err, "creating gzip readere for state snapshot data")
	}
	defer utils.IgnoreError(gzReader.Close)

	allObjBytes, err := ioutil.ReadAll(gzReader)
	if err != nil {
		return nil, errors.Wrap(err, "reading compressed state snapshot data")
	}

	objBytes := bytes.Split(allObjBytes, jsonSeparator)

	universalDeserializer := s.ctx.UniversalDecoder()

	result := make([]k8sobjects.Object, 0, len(objBytes))
	for _, serialized := range objBytes {
		runtimeObj, _, err := universalDeserializer.Decode(serialized, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not deserialize object in stored snapshot")
		}
		obj, _ := runtimeObj.(k8sobjects.Object)
		if obj == nil {
			return nil, errors.Errorf("object of kind %v does not have object metadata", runtimeObj.GetObjectKind().GroupVersionKind())
		}
		result = append(result, obj)
	}

	return result, nil
}

func (s *snapshotter) createStateSnapshot() ([]k8sobjects.Object, *v1.Secret, error) {
	objs, err := s.listResources()
	if err != nil {
		return nil, nil, err
	}

	byteSlices := make([][]byte, 0, len(objs))
	jsonSerializer := json.NewSerializer(json.DefaultMetaFactory, nil, nil, false)
	for _, obj := range objs {
		var buf bytes.Buffer
		if err := jsonSerializer.Encode(obj, &buf); err != nil {
			return nil, nil, errors.Wrapf(err, "marshaling object of kind %v to JSON", obj.GetObjectKind().GroupVersionKind())
		}
		byteSlices = append(byteSlices, buf.Bytes())
	}

	var compressedData bytes.Buffer
	gzipWriter, err := gzip.NewWriterLevel(&compressedData, gzip.BestCompression)
	if err != nil {
		return nil, nil, errorhelpers.PanicOnDevelopment(err) // level is valid, so expect no error
	}
	if _, err := gzipWriter.Write(bytes.Join(byteSlices, jsonSeparator)); err != nil {
		return nil, nil, errorhelpers.PanicOnDevelopment(err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, nil, errorhelpers.PanicOnDevelopment(err)
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.Namespace,
			Name:      secretName,
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			secretDataName: compressedData.Bytes(),
		},
	}
	s.ctx.AnnotateProcessStateObject(secret)

	return objs, secret, nil
}

func (s *snapshotter) unpackList(listObj runtime.Object) ([]k8sobjects.Object, error) {
	objs, ok := unpackListReflect(listObj)
	if ok {
		return objs, nil
	}

	log.Infof("Could not unpack list of kind %v using reflection", listObj.GetObjectKind().GroupVersionKind())

	var list unstructured.UnstructuredList
	if err := s.ctx.Scheme().Convert(listObj, &list, nil); err != nil {
		return nil, errors.Wrapf(err, "converting object of kind %v to a generic list", listObj.GetObjectKind().GroupVersionKind())
	}

	objs = make([]k8sobjects.Object, 0, len(list.Items))
	for _, item := range list.Items {
		objs = append(objs, &item)
	}
	return objs, nil
}

func (s *snapshotter) listResources() ([]k8sobjects.Object, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", common.UpgradeResourceLabelKey, common.UpgradeResourceLabelValue),
	}

	var result []k8sobjects.Object

	for _, resourceType := range s.ctx.Resources() {
		resourceClient, err := s.ctx.DynamicClientForResource(resourceType, common.Namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "obtaining dynamic client for resource %v", resourceType)
		}
		listObj, err := resourceClient.List(listOpts)
		if err != nil {
			return nil, errors.Wrapf(err, "listing relevant objects of type %v", resourceType)
		}

		objs, err := s.unpackList(listObj)
		if err != nil {
			return nil, errors.Wrapf(err, "unpacking list of objects of type %v", resourceType)
		}
		result = append(result, objs...)
	}

	return result, nil
}
