#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
KUBE_COMMAND=${KUBE_COMMAND:-{{.K8sConfig.Command}}}
NAMESPACE="${ROX_NAMESPACE:-stackrox}"

echo "Deploy Central DB ..."
${KUBE_COMMAND} -n ${NAMESPACE} apply -f "${DIR}/central/."
${KUBE_COMMAND} -n ${NAMESPACE} patch deploy/central -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [
          {
            "name": "central",
            "volumeMounts": [
              {
                "name": "central-db-password",
                "mountPath": "/run/secrets/stackrox.io/db-password"
              },
              {
                "name": "central-external-db-volume",
                "mountPath": "/etc/ext-db"
              }
            ]
          }
        ],
        "volumes": [
          {
            "name": "central-db-password",
            "secret": {
              "secretName": "central-db-password"
            }
          },
          {
            "name": "central-external-db-volume",
            "configMap": {
              "name": "central-external-db",
              "optional": true
            }
          }
        ]
      }
    }
  }
}
'
