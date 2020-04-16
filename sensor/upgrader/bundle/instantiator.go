package bundle

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type instantiator struct {
	ctx *upgradectx.UpgradeContext
}

func (i *instantiator) Instantiate(bundleContents Contents) ([]k8sutil.Object, error) {
	trackedBundleContents := trackContents(bundleContents)

	var allObjects []k8sutil.Object
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
	neglectedFiles.RemoveMatching(common.IsWhitelistedBundleFile)

	if neglectedFiles.Cardinality() > 0 {
		return nil, errors.Errorf("the following non-whitelisted files in the bundle have been neglected: %s", neglectedFiles.ElementsString(", "))
	}

	if err := validateMetadata(allObjects); err != nil {
		return nil, err
	}
	return allObjects, nil
}

func (i *instantiator) loadObjectsFromYAMLs(c Contents) ([]k8sutil.Object, error) {
	var result []k8sutil.Object
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

func (i *instantiator) readObjectFromYAMLReader(r *yaml.YAMLReader) (k8sutil.Object, error) {
	doc, err := r.Read()
	if err != nil {
		return nil, err
	}

	obj, err := i.ctx.ParseAndValidateObject(doc)
	if err != nil {
		return nil, errors.Wrapf(err, "decoding document %s", string(doc))
	}
	return obj, nil
}

func (i *instantiator) loadObjectsFromYAML(openFn func() (io.ReadCloser, error)) ([]k8sutil.Object, error) {
	reader, err := openFn()
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(reader.Close)

	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var objects []k8sutil.Object

	yamlReader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewBuffer(contents)))
	for {
		var obj k8sutil.Object
		obj, err = i.readObjectFromYAMLReader(yamlReader)
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

func validateMetadata(objs []k8sutil.Object) error {
	errs := errorhelpers.NewErrorList("object metadata validation failed")
	for _, obj := range objs {
		if labelVal := obj.GetLabels()[common.UpgradeResourceLabelKey]; labelVal != common.UpgradeResourceLabelValue {
			errs.AddStringf("upgrade label %s of object %s has invalid value %q, expected: %q", common.UpgradeResourceLabelKey, k8sobjects.RefOf(obj), labelVal, common.UpgradeResourceLabelValue)
		}
	}
	return errs.ToError()
}
