package service

import (
	"testing"

	openshiftAppsV1 "github.com/openshift/api/apps/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

const openshiftDeploymentConfigYaml = `
apiVersion: apps.openshift.io/v1
kind: DeploymentConfig
metadata:
  name: frontend
  namespace: frontend
  labels:
    app: frontend
spec:
  replicas: 5
  selector:
    app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - image: hello-openshift:latest
        name: helloworld
        ports:
        - containerPort: 8080
          protocol: TCP
        restartPolicy: Always
  triggers:
  - type: ConfigChange 
  - imageChangeParams:
      automatic: true
      containerNames:
      - helloworld
      from:
        kind: ImageStreamTag
        name: hello-openshift:latest
    type: ImageChange  
  strategy:
    type: Rolling
`

const multiYaml = `
apiVersion: v1
kind: Service
metadata:
  name: wordpress
  labels:
    app: wordpress
spec:
  ports:
    - port: 22
  selector:
    app: wordpress
    tier: frontend
  type: LoadBalancer
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: wp-pv-claim
  labels:
    app: wordpress
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
---
apiVersion: apps/v1 # for versions before 1.9.0 use apps/v1beta2
kind: Deployment
metadata:
  name: wordpress
  labels:
    app: wordpress
spec:
  selector:
    matchLabels:
      app: wordpress
      tier: frontend
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: wordpress
        tier: frontend
    spec:
      containers:
      - image: wordpress:latest
        name: wordpress
        env:
        - name: WORDPRESS_DB_HOST
          value: wordpress-mysql
        - name: WORDPRESS_DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mysql-pass
              key: password
        ports:
        - containerPort: 22
          name: wordpress
        volumeMounts:
        - name: wordpress-persistent-storage
          mountPath: /var/www/html
      volumes:
      - name: wordpress-persistent-storage
        persistentVolumeClaim:
          claimName: wp-pv-claim
---
apiVersion: apps.openshift.io/v1
kind: DeploymentConfig
metadata:
  name: frontend
  namespace: frontend
  labels:
    app: frontend
spec:
  replicas: 5
  selector:
    app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - image: hello-openshift:latest
        name: helloworld
        ports:
        - containerPort: 8080
          protocol: TCP
        restartPolicy: Always
  triggers:
  - type: ConfigChangeg
  - imageChangeParams:
      automatic: true
      containerNames:
      - helloworld
      from:
        kind: ImageStreamTag
        name: hello-openshift:latest
    type: ImageChange
  strategy:
    type: Rolling
---
apiVersion: v1
kind: DeploymentConfig
metadata:
  name: frontend
  namespace: frontend
  labels:
    app: frontend
spec:
  replicas: 5
  selector:
    app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - image: hello-openshift:latest
        name: helloworld
        ports:
        - containerPort: 8080
          protocol: TCP
        restartPolicy: Always
  triggers:
  - type: ConfigChange 
  - imageChangeParams:
      automatic: true
      containerNames:
      - helloworld
      from:
        kind: ImageStreamTag
        name: hello-openshift:latest
    type: ImageChange  
  strategy:
    type: Rolling
`

const unregisteredType = `
apiVersion: apps/v1
kind: DeploymentConfig
metadata:
  name: frontend
  namespace: frontend
  labels:
    app: frontend
spec:
  replicas: 5
  selector:
    app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - image: hello-openshift:latest
        name: helloworld
        ports:
        - containerPort: 8080
          protocol: TCP
        restartPolicy: Always
  triggers:
  - type: ConfigChange
  - imageChangeParams:
      automatic: true
      containerNames:
      - helloworld
      from:
        kind: ImageStreamTag
        name: hello-ogpenshift:latest
    type: ImageChange
  strategy:
    type: Rolling
`

const openshiftDeployConfMultiYaml = `
apiVersion: apps.openshift.io/v1
kind: DeploymentConfig
metadata:
  name: frontend
  namespace: frontend
  labels:
    app: frontend
spec:
  replicas: 5
  selector:
    app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - image: hello-openshift:latest
        name: helloworld
        ports:
        - containerPort: 8080
          protocol: TCP
        restartPolicy: Always
  triggers:
  - type: ConfigChange
  - imageChangeParams:
      automatic: true
      containerNames:
      - helloworld
      from:
        kind: ImageStreamTag
        name: hello-ogpenshift:latest
    type: ImageChange
  strategy:
    type: Rolling
---
apiVersion: v1
kind: DeploymentConfig
metadata:
  name: frontend
  namespace: frontend
  labels:
    app: frontend
spec:
  replicas: 5
  selector:
    app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - image: hello-openshift:latest
        name: helloworld
        ports:
        - containerPort: 8080
          protocol: TCP
        restartPolicy: Always
  triggers:
  - type: ConfigChange
  - imageChangeParams:
      automatic: true
      containerNames:
      - helloworld
      from:
        kind: ImageStreamTag
        name: hello-ogpenshift:latest
    type: ImageChange
  strategy:
    type: Rolling
`

func TestParseList_Success(t *testing.T) {
	_, err := getObjectsFromYAML(listYAML)
	require.NoError(t, err)

	_, err = getObjectsFromYAML(openshiftDeploymentConfigYaml)
	require.NoError(t, err)

	_, err = getObjectsFromYAML(multiYaml)
	require.NoError(t, err)

	_, err = getObjectsFromYAML(openshiftDeploymentConfigYaml)
	require.NoError(t, err)
}

func TestParseList_Error(t *testing.T) {
	_, err := getObjectsFromYAML(unregisteredType)
	require.Error(t, err)
}

func TestParseList_ConversionToOpenshiftObjects(t *testing.T) {
	objs, err := getObjectsFromYAML(openshiftDeployConfMultiYaml)
	require.NoError(t, err)

	for i := range objs {
		obj := objs[i]
		assert.IsType(t, (*openshiftAppsV1.DeploymentConfig)(nil), obj)
	}
}
