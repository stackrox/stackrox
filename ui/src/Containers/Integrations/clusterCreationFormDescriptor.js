export const k8sCreationFormDescriptor = [
    {
        label: 'Name',
        type: 'text',
        jsonpath: 'name',
        placeholder: 'Cluster name',
        disabled: false
    },
    {
        label: 'Image name (Prevent location)',
        type: 'text',
        jsonpath: 'preventImage',
        placeholder: 'stackrox.io/prevent:[current-version]',
        disabled: false
    },
    {
        label: 'Central API Endpoint',
        type: 'text',
        jsonpath: 'centralApiEndpoint',
        placeholder: 'central.stackrox:443',
        disabled: false
    },
    {
        label: 'Namespace',
        type: 'text',
        jsonpath: 'namespace',
        placeholder: 'stackrox',
        disabled: false
    },
    {
        label: 'Image Pull Secret Name',
        type: 'text',
        jsonpath: 'imagePullSecret',
        placeholder: 'stackrox',
        disabled: false
    }
];

export const openshiftCreationFormDescriptor = [
    {
        label: 'Name',
        type: 'text',
        jsonpath: 'name',
        placeholder: 'Cluster name',
        disabled: false
    },
    {
        label: 'Image name (Prevent location)',
        type: 'text',
        jsonpath: 'preventImage',
        placeholder: 'docker-registry.default.svc:5000/stackrox/prevent:[current-version]',
        disabled: false
    },
    {
        label: 'Central API Endpoint',
        type: 'text',
        jsonpath: 'centralApiEndpoint',
        placeholder: 'central.stackrox:443',
        disabled: false
    },
    {
        label: 'Namespace',
        type: 'text',
        jsonpath: 'namespace',
        placeholder: 'stackrox',
        disabled: false
    }
];

export const dockerClusterCreationFormDescriptor = [
    {
        label: 'Name',
        type: 'text',
        jsonpath: 'name',
        placeholder: 'Cluster name',
        disabled: false
    },
    {
        label: 'Image name (Prevent location)',
        type: 'text',
        jsonpath: 'preventImage',
        placeholder: 'stackrox.io/prevent:[current-version]',
        disabled: false
    },
    {
        label: 'Central API Endpoint',
        type: 'text',
        jsonpath: 'centralApiEndpoint',
        placeholder: 'central.prevent_net:443',
        disabled: false
    },
    {
        label: 'Disable Swarm TLS',
        type: 'checkbox',
        jsonpath: 'swarm.disableSwarmTls',
        disabled: false
    }
];
