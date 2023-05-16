# Prototype - Helm only mode

Goal of the prototype is to be independent of the CRD schema but leveraging the Helm reconcilers capabilities to 
manage Helm charts.

Sketch:
 - Add Annotation for schemaless mode
 - Annotation for Helm values to overwrite
 - Disable extensions and translator (everything additional to the Helm reconcile)

Extensions:

 - PVC can be disabled
 - TLS and password secrets not deleted after CR deletion
 - 

## CRD

```
kind: Central
annotations:
    "stackrox.io/version": "v3.80.0" 
    "stackrox.io/no-crd": "true"
metadata:
    name: some-service
spec:
    values: |
        // ...
```
