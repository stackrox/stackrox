package controller

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	pod                   string = "Pod"
	replicaSet            string = "ReplicaSet"
	replicationController string = "ReplicationController"
	deployment            string = "Deployment"
	statefulset           string = "StatefulSet"
	daemonset             string = "DaemonSet"
	job                   string = "Job"
	cronJob               string = "CronTab"
	service               string = "Service"
	configmap             string = "ConfigMap"
)

var (
	acceptedK8sTypesRegex = fmt.Sprintf("(^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$)",
		pod, replicaSet, replicationController, deployment, daemonset, statefulset, job, cronJob, service, configmap)
	acceptedK8sTypes = regexp.MustCompile(acceptedK8sTypesRegex)
	yamlSuffix       = regexp.MustCompile(".ya?ml$")
)

type parsedK8sObjects struct {
	ManifestFilepath string
	DeployObjects    []deployObject
}

type deployObject struct {
	GroupKind     string
	RuntimeObject []byte
}

// return a list of yaml files under a given directory (recursively)
func searchDeploymentManifests(repoDir string, stopOn1stErr bool) ([]string, []FileProcessingError) {
	yamls := []string{}
	errors := []FileProcessingError{}
	err := filepath.WalkDir(repoDir, func(path string, f os.DirEntry, err error) error {
		if err != nil {
			errors = appendAndLogNewError(errors, failedAccessingDir(path, err, path != repoDir))
			if stopProcessing(stopOn1stErr, errors) {
				return err
			}
			return filepath.SkipDir
		}
		if f != nil && !f.IsDir() && yamlSuffix.MatchString(f.Name()) {
			yamls = append(yamls, path)
		}
		return nil
	})
	if err != nil {
		activeLogger.Errorf(err, "Error walking directory")
	}
	return yamls, errors
}

func getK8sDeploymentResources(repoDir string, stopOn1stErr bool) ([]parsedK8sObjects, []FileProcessingError) {
	manifestFiles, fileScanErrors := searchDeploymentManifests(repoDir, stopOn1stErr)
	if stopProcessing(stopOn1stErr, fileScanErrors) {
		return nil, fileScanErrors
	}
	if len(manifestFiles) == 0 {
		fileScanErrors = appendAndLogNewError(fileScanErrors, noYamlsFound())
		return nil, fileScanErrors
	}

	parsedObjs := []parsedK8sObjects{}
	for _, mfp := range manifestFiles {
		deployObjects, err := parseK8sYaml(mfp, stopOn1stErr)
		fileScanErrors = append(fileScanErrors, err...)
		if stopProcessing(stopOn1stErr, fileScanErrors) {
			return nil, fileScanErrors
		}
		if len(deployObjects) > 0 {
			manifestFilepath := mfp
			if pathSplit := strings.Split(mfp, repoDir); len(pathSplit) > 1 {
				manifestFilepath = pathSplit[1]
			}
			parsedObjs = append(parsedObjs, parsedK8sObjects{DeployObjects: deployObjects, ManifestFilepath: manifestFilepath})
		}
	}
	return parsedObjs, fileScanErrors
}

func splitByYamlDocuments(mfp string) ([]string, []FileProcessingError) {
	fileBuf, err := os.ReadFile(mfp)
	if err != nil {
		return []string{}, appendAndLogNewError(nil, failedReadingFile(mfp, err))
	}

	decoder := yaml.NewDecoder(bytes.NewBuffer(fileBuf))
	documents := []string{}
	documentID := 0
	for {
		var doc yaml.Node
		if err := decoder.Decode(&doc); err != nil {
			if err != io.EOF {
				return documents, appendAndLogNewError(nil, malformedYamlDoc(mfp, 0, documentID, err))
			}
			break
		}
		if len(doc.Content) > 0 && doc.Content[0].Kind == yaml.MappingNode {
			out, err := yaml.Marshal(doc.Content[0])
			if err != nil {
				return documents, appendAndLogNewError(nil, malformedYamlDoc(mfp, doc.Line, documentID, err))
			}
			documents = append(documents, string(out))
		}
		documentID += 1
	}
	return documents, nil
}

func parseK8sYaml(mfp string, stopOn1stErr bool) ([]deployObject, []FileProcessingError) {
	dObjs := []deployObject{}
	sepYamlFiles, fileProcessingErrors := splitByYamlDocuments(mfp)
	if stopProcessing(stopOn1stErr, fileProcessingErrors) {
		return nil, fileProcessingErrors
	}

	for docID, doc := range sepYamlFiles {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		_, groupVersionKind, err := decode([]byte(doc), nil, nil)
		if err != nil {
			fileProcessingErrors = appendAndLogNewError(fileProcessingErrors, notK8sResource(mfp, docID, err))
			continue
		}
		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			activeLogger.Infof("in file: %s, document: %d, skipping object with type: %s", mfp, docID, groupVersionKind.Kind)
		} else {
			d := deployObject{}
			d.GroupKind = groupVersionKind.Kind
			d.RuntimeObject = []byte(doc)
			dObjs = append(dObjs, d)
		}
	}
	return dObjs, fileProcessingErrors
}

func stopProcessing(stopOn1stErr bool, errs []FileProcessingError) bool {
	for idx := range errs {
		if errs[idx].IsFatal() || stopOn1stErr && errs[idx].IsSevere() {
			return true
		}
	}

	return false
}

func appendAndLogNewError(errs []FileProcessingError, newErr *FileProcessingError) []FileProcessingError {
	logError(newErr)
	errs = append(errs, *newErr)
	return errs
}
