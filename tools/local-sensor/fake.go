package main

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var clusterResources = []string{
	"Namespace",
	"ClusterRole",
	"ClusterRoleBinding",
	"Node",
}

func isClusterScope(kind string) bool {
	for _, r := range clusterResources {
		if r == kind {
			return true
		}
	}
	return false
}

func getAPIVersion(obj map[string]interface{}) (string, error) {
	metaData, ok := obj["metadata"]
	if !ok {
		return "", errors.New("'metadata' key not found in resource")
	}
	obj, ok = metaData.(map[string]interface{})
	if !ok {
		return "", errors.New("'metadata' cast error")
	}
	managedFields, ok := obj["managedFields"]
	if !ok {
		return "", errors.New("'managedFields' key not found in 'metadata'")
	}
	objSlice, ok := managedFields.([]interface{})
	if !ok {
		return "", errors.New("cannot cast 'managedFields' slice")
	}
	obj, ok = objSlice[0].(map[string]interface{})
	if !ok {
		return "", errors.New("'managedFields' cast error")
	}
	apiVersion, ok := obj["apiVersion"]
	if !ok {
		return "", errors.New("'apiVersion' not found in 'managedFields'")
	}
	apiVersionStr, ok := apiVersion.(string)
	if !ok {
		return "", errors.New("'apiVersion' cast error")
	}
	return apiVersionStr, nil
}

func createEvent(client *k8s.ClientSet, msg resources.InformerK8sMsg) error {
	obj := &unstructured.Unstructured{}
	o, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return errors.New("'Payload' cast error")
	}
	obj.Object = o
	objType := strings.Split(msg.ObjectType, ".")
	obj.SetKind(objType[1])
	obj.SetAPIVersion(objType[0])
	if apiVersion, err := getAPIVersion(o); err != nil {
		log.Printf("resouce '%s': %s", msg.ObjectType, err)
	} else {
		obj.SetAPIVersion(apiVersion)
	}

	payload, err := obj.MarshalJSON()
	if err != nil {
		return err
	}
	_, gvk, err := unstructured.UnstructuredJSONScheme.Decode(payload, nil, obj)
	if err != nil {
		return err
	}

	resource := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  obj.GetAPIVersion(),
		Resource: obj.GetKind(),
	}

	var dr dynamic.ResourceInterface
	if isClusterScope(obj.GetKind()) {
		dr = client.Dynamic().Resource(resource)
	} else {
		dr = client.Dynamic().Resource(resource).Namespace(obj.GetNamespace())
	}

	switch msg.Action {
	case "CREATE_RESOURCE":
		if _, err := dr.Create(context.Background(), obj, metav1.CreateOptions{}); err != nil {
			if k8sErrors.IsAlreadyExists(err) {
				if _, err := dr.Update(context.Background(), obj, metav1.UpdateOptions{}); err != nil {
					log.Printf("cannot update event for %s %s", msg.ObjectType, err)
					return err
				}
			} else {
				log.Printf("cannot create event for %s %s", msg.ObjectType, err)
				return err
			}
		}
		break
	case "UPDATE_RESOURCE":
		if _, err := dr.Update(context.Background(), obj, metav1.UpdateOptions{}); err != nil {
			if k8sErrors.IsNotFound(err) {
				if _, err := dr.Create(context.Background(), obj, metav1.CreateOptions{}); err != nil {
					log.Printf("cannot create event for %s %s", msg.ObjectType, err)
					return err
				}
			} else {
				log.Printf("cannot update event for %s %s", msg.ObjectType, err)
				return err
			}
		}
		break
	case "DELETE_RESOURCE":
		if err := dr.Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{}); err != nil {
			log.Printf("cannot delete event for %s %s", msg.ObjectType, err)
			return err
		}
		break
	}
	return nil
}
