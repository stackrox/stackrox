package bundle

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type instantiator struct {
	ctx upgradeContext
}

func (i *instantiator) Instantiate(bundleContents Contents) ([]*unstructured.Unstructured, error) {
	trackedBundleContents := trackContents(bundleContents)

	var allObjects []*unstructured.Unstructured
	dynamicObjs, err := createDynamicObjects(trackedBundleContents)
	if err != nil {
		return nil, errors.Wrap(err, "creating config objects from sensor bundle")
	}
	allObjects = append(allObjects, dynamicObjs...)

	yamlObjs, err := i.loadObjectsFromYAMLs(trackedBundleContents)
	if err != nil {
		return nil, errors.Wrap(err, "loading objects from sensor bundle YAMLs")
	}
	allObjects = append(allObjects, yamlObjs...)

	neglectedFiles := set.NewStringSet(bundleContents.ListFiles()...)
	for _, openedFile := range trackedBundleContents.OpenedFiles() {
		neglectedFiles.Remove(openedFile)
	}
	neglectedFiles.RemoveMatching(common.IsIgnorelistedBundleFile)

	if neglectedFiles.Cardinality() > 0 {
		return nil, errors.Errorf("the following un-ignored files in the bundle have been neglected: %s", neglectedFiles.ElementsString(", "))
	}

	// Remove the additional-ca-sensor secret.
	common.Filter(&allObjects, common.Not(common.AdditionalCASecretPredicate))

	if err := validateMetadata(allObjects); err != nil {
		return nil, err
	}

	if i.ctx.InCertRotationMode() {
		common.Filter(&allObjects, common.CertObjectPredicate)
	}

	// Remove the psps if not enabled
	if !i.ctx.IsPodSecurityEnabled() {
		common.Filter(&allObjects, common.Not(common.PSPObjectPredicate))
	}

	return allObjects, nil
}

func (i *instantiator) loadObjectsFromYAMLs(c Contents) ([]*unstructured.Unstructured, error) {
	var result []*unstructured.Unstructured
	for _, fileName := range c.ListFiles() {
		if !strings.HasSuffix(fileName, ".yaml") {
			continue
		}

		fileObjs, err := i.loadObjectsFromYAML(c.File(fileName))
		if err != nil {
			return nil, errors.Wrapf(err, "loading objects from YAML file %s", fileName)
		}

		result = append(result, fileObjs...)
	}
	return result, nil
}

func (i *instantiator) loadObjectsFromYAML(openFn func() (io.ReadCloser, error)) ([]*unstructured.Unstructured, error) {
	reader, err := openFn()
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(reader.Close)

	contents, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var objects []*unstructured.Unstructured

	yamlReader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewBuffer(contents)))
	yamlDoc, err := yamlReader.Read()
	for ; err == nil; yamlDoc, err = yamlReader.Read() {
		// First, test if the document is empty. We cannot simply trim spaces and check for an empty slice,
		// as it could contain comments.
		var rawObj interface{}
		if err := yaml.Unmarshal(yamlDoc, &rawObj); err != nil {
			return nil, errors.Wrap(err, "invalid YAML in multi-document file")
		}
		if rawObj == nil {
			continue
		}

		// Then, decode it as a Kubernetes object.
		var obj *unstructured.Unstructured
		obj, err = i.ctx.ParseAndValidateObject(yamlDoc)
		if err != nil {
			break
		}
		objects = append(objects, obj)
	}

	if err != io.EOF {
		return nil, err
	}

	return objects, nil
}

func validateMetadata(objs []*unstructured.Unstructured) error {
	errs := errorhelpers.NewErrorList("object metadata validation failed")
	for i := range objs {
		obj := objs[i]
		if labelVal := obj.GetLabels()[common.UpgradeResourceLabelKey]; labelVal != common.UpgradeResourceLabelValue {
			errs.AddStringf("upgrade label %s of object %s has invalid value %q, expected: %q", common.UpgradeResourceLabelKey, k8sobjects.RefOf(obj), labelVal, common.UpgradeResourceLabelValue)
		}
	}
	return errs.ToError()
}
