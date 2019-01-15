package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

const listYAML = `
apiVersion: v1
items:
- apiVersion: extensions/v1beta1
  kind: Deployment
  metadata:
    annotations:
      deployment.kubernetes.io/revision: "1"
      email: support@stackrox.com
      owner: stackrox
    creationTimestamp: 2018-12-19T23:31:01Z
    generation: 1
    labels:
      app: central
    name: central
    namespace: stackrox
    resourceVersion: "4265631"
    selfLink: /apis/extensions/v1beta1/namespaces/stackrox/deployments/central
    uid: 2582c24a-03e6-11e9-a2fd-025000000001
  spec:
    minReadySeconds: 15
    progressDeadlineSeconds: 600
    replicas: 1
    revisionHistoryLimit: 10
    selector:
      matchLabels:
        app: central
    strategy:
      rollingUpdate:
        maxSurge: 1
        maxUnavailable: 1
      type: RollingUpdate
    template:
      metadata:
        creationTimestamp: null
        labels:
          app: central
        namespace: stackrox
      spec:
        containers:
        - command:
          - /stackrox/entrypoint.sh
          - central
          env:
          - name: ROX_HTPASSWD_AUTH
            value: "true"
          image: stackrox/main:2.3.14.0-9-g80590ca285-dirty
          imagePullPolicy: IfNotPresent
          name: central
          ports:
          - containerPort: 443
            name: api
            protocol: TCP
          resources:
            limits:
              cpu: "2"
              memory: 8Gi
            requests:
              cpu: "1"
              memory: 2Gi
          securityContext:
            capabilities:
              drop:
              - NET_RAW
            readOnlyRootFilesystem: true
          terminationMessagePath: /dev/termination-log
          terminationMessagePolicy: File
          volumeMounts:
          - mountPath: /var/log/stackrox/
            name: varlog
          - mountPath: /run/secrets/stackrox.io/certs/
            name: central-certs-volume
            readOnly: true
          - mountPath: /run/secrets/stackrox.io/htpasswd/
            name: central-htpasswd-volume
            readOnly: true
          - mountPath: /run/secrets/stackrox.io/jwt/
            name: central-jwt-volume
            readOnly: true
          - mountPath: /usr/local/share/ca-certificates/
            name: additional-ca-volume
            readOnly: true
          - mountPath: /var/lib/stackrox
            name: empty-db
          - mountPath: /run/secrets/stackrox.io/monitoring/certs
            name: monitoring-client-volume
            readOnly: true
        - command:
          - /telegraf
          env:
          - name: SERVICE
            value: central
          - name: CLUSTER_NAME
            value: main
          - name: PROMETHEUS_ENDPOINT
            value: https://localhost:443
          - name: MONITORING_ENDPOINT
            value: monitoring.stackrox:443
          image: stackrox/main:2.3.14.0-9-g80590ca285-dirty
          imagePullPolicy: IfNotPresent
          name: telegraf
          resources:
            limits:
              cpu: 100m
              memory: 100Mi
            requests:
              cpu: 50m
              memory: 50Mi
          securityContext:
            capabilities:
              drop:
              - NET_RAW
          terminationMessagePath: /dev/termination-log
          terminationMessagePolicy: File
          volumeMounts:
          - mountPath: /var/log/stackrox/
            name: varlog
            readOnly: true
          - mountPath: /run/secrets/stackrox.io/monitoring/certs
            name: monitoring-client-volume
            readOnly: true
          - mountPath: /etc/telegraf/
            name: telegraf-config-volume
            readOnly: true
        dnsPolicy: ClusterFirst
        restartPolicy: Always
        schedulerName: default-scheduler
        securityContext: {}
        serviceAccount: central
        serviceAccountName: central
        terminationGracePeriodSeconds: 30
        volumes:
        - emptyDir: {}
          name: varlog
        - name: central-certs-volume
          secret:
            defaultMode: 420
            secretName: central-tls
        - name: central-htpasswd-volume
          secret:
            defaultMode: 420
            optional: true
            secretName: central-htpasswd
        - name: central-jwt-volume
          secret:
            defaultMode: 420
            items:
            - key: jwt-key.der
              path: jwt-key.der
            secretName: central-tls
        - name: additional-ca-volume
          secret:
            defaultMode: 420
            optional: true
            secretName: additional-ca
        - name: monitoring-client-volume
          secret:
            defaultMode: 420
            secretName: monitoring-client
        - configMap:
            defaultMode: 420
            name: telegraf
          name: telegraf-config-volume
        - emptyDir: {}
          name: empty-db
  status:
    availableReplicas: 1
    conditions:
    - lastTransitionTime: 2018-12-19T23:31:01Z
      lastUpdateTime: 2018-12-19T23:31:01Z
      message: Deployment has minimum availability.
      reason: MinimumReplicasAvailable
      status: "True"
      type: Available
    - lastTransitionTime: 2018-12-19T23:31:01Z
      lastUpdateTime: 2018-12-19T23:31:24Z
      message: ReplicaSet "central-5b85d56f5c" has successfully progressed.
      reason: NewReplicaSetAvailable
      status: "True"
      type: Progressing
    observedGeneration: 1
    readyReplicas: 1
    replicas: 1
    updatedReplicas: 1
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
`

func TestParseList(t *testing.T) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(listYAML), nil, nil)
	require.NoError(t, err)
	if list, ok := obj.(*v1.List); ok {
		for _, i := range list.Items {
			_, _, err := decode(i.Raw, nil, nil)
			require.NoError(t, err)
		}
	}
}
