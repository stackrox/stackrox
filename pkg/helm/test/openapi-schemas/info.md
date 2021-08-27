## Schema sources

- OpenShift 3.11 schema source: https://github.com/garethr/openshift-json-schema/blob/master/v3.11.0/_definitions.json
- OpenShift 4.1 schema source: https://github.com/garethr/openshift-json-schema/blob/master/v4.1.0/_definitions.json

## Adhoc generation

If there are no JSON schemas available the CRD can be used to fill the
gap. e.g. see com.coreos.json.gz, the definition can be largely derived from:

```
oc get crd servicemonitors.monitoring.coreos.com -o json | jq '.spec.versions[0].schema.openAPIV3Schema'
```

with an addition of a "x-kubernetes-group-version-kind" to map to the API ref:

```
"x-kubernetes-group-version-kind": [
  {
    "group": "monitoring.coreos.com",
    "kind": "ServiceMonitor",
    "version": "v1"
  }
]
```