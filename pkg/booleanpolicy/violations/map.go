package violations

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

func extractMapKV(e string) (string, string, error) {
	kvPair := strings.SplitN(e, augmentedobjs.CompositeFieldCharSep, 2)
	if len(kvPair) != 2 {
		return "", "", errors.New("invalid key value pair in map result")
	}
	return kvPair[0], kvPair[1], nil
}

func getResourceNameAndKVSentence(baseResourceName string, keyValues []string) (string, string) {
	keyValueMatches := make([]string, 0, len(keyValues))
	for _, keyValue := range keyValues {
		key, value, err := extractMapKV(keyValue)
		if err != nil || key == "" {
			continue
		}
		keyValueMatches = append(keyValueMatches, fmt.Sprintf("'%s: %s'", key, value))
	}
	resourceName := baseResourceName
	if len(keyValueMatches) == 0 {
		resourceName = "no " + resourceName
	}
	if len(keyValueMatches) != 1 {
		resourceName = resourceName + "s"
	}
	return resourceName, stringSliceToSortedSentence(keyValueMatches)
}

func mapPrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	msgTemplate := `{{.Object}} includes {{.Resource}}{{if .KVs}} {{.KVs}}{{end}}`
	type resultFields struct {
		Object   string
		Resource string
		KVs      string
	}
	r := make([]resultFields, 0)
	if annotations, ok := fieldMap[search.Annotation.String()]; ok {
		resourceName, KVs := getResourceNameAndKVSentence("annotation", annotations)
		r = append(r, resultFields{Resource: resourceName, KVs: KVs, Object: "Deployment"})
	}
	if labels, ok := fieldMap[search.Label.String()]; ok {
		resourceName, KVs := getResourceNameAndKVSentence("label", labels)
		r = append(r, resultFields{Resource: resourceName, KVs: KVs, Object: "Deployment"})
	}
	if imageLabels, ok := fieldMap[search.ImageLabel.String()]; ok {
		resourceName, KVs := getResourceNameAndKVSentence("label", imageLabels)
		r = append(r, resultFields{Resource: resourceName, KVs: KVs, Object: "Image"})
	}

	messages := make([]string, 0, len(r))
	for _, values := range r {
		msg, err := executeTemplate(msgTemplate, values)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg...)
	}
	return messages, nil
}
