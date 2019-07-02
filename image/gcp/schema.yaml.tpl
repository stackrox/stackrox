applicationApiVersion: v1beta1

properties:
  # Required solution properties.
  name:
    type: string
    default: stackrox
    x-google-marketplace:
      type: NAME

  namespace:
    type: string
    default: stackrox
    x-google-marketplace:
      type: NAMESPACE

  main-image:
    type: string
    title: Stackrox image name
    description: Name of Stackrox image to use
    default: gcr.io/stackrox-launcher-project-1/stackrox:$MAIN_IMAGE_TAG
    x-google-marketplace:
      type: IMAGE

  monitoring-image:
    type: string
    title: Monitoring image to use
    description: Monitoring image to use
    default: gcr.io/stackrox-launcher-project-1/stackrox/monitoring:$MAIN_IMAGE_TAG
    x-google-marketplace:
      type: IMAGE

  scanner-image:
    type: string
    title: Stackrox Scanner image name
    description: Name of Stackrox scanner image to use
    default: gcr.io/stackrox-launcher-project-1/stackrox/scanner:$SCANNER_IMAGE_TAG
    x-google-marketplace:
      type: IMAGE

  # Secrets.
  license:
    type: string
    title: License key
    description: Text of the Stackrox license key

  password:
    type: string
    title: Admin password
    description: Stackrox administrator password

  # Networking.
  lb-type:
    type: string
    title: the method of exposing Central (lb, np, none)
    description: the method of exposing Central (lb, np, none)
    default: none
    enum:
      - lb
      - np
      - none

#  offline:
#    type: boolean
#    title: run StackRox in offline mode which avoids reaching out to the internet
#    description: run StackRox in offline mode which avoids reaching out to the internet
#    default: false

  # Storage
#  name:
#    type: string
#    title: external volume name
#    description: external volume name
#    default: stackrox-db

#  size:
#    type: integer
#    title: external volume size in Gi (optional, defaults to 100Gi)
#    description: external volume size in Gi (optional, defaults to 100Gi)
#    default: "100"

#  storage-class:
#    type: string
#    title: storage class name (optional if you have a default StorageClass configured)
#    description: storage class name (optional if you have a default StorageClass configured)

  svcacct:
    type: string
    title: StackRox Deployer Service Account
    description: Service account used by the Deployer to install StackRox
    x-google-marketplace:
      type: SERVICE_ACCOUNT
      serviceAccount:
        roles:
        - type: ClusterRole
          rulesType: CUSTOM
          rules:
          - apiGroups: ['*']
            resources: ['*']
            verbs: ['*']

required:
- name
- namespace
- main-image
- scanner-image
- monitoring-image
- license
- password
- lb-type
- svcacct
