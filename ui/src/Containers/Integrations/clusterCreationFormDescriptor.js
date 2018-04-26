export const clusterCreationFormDescriptor = [
    {
        label: 'Name',
        type: 'text',
        value: 'name',
        placeholder: 'Cluster name',
        disabled: false
    },
    {
        label: 'Image name (Prevent location)',
        type: 'text',
        value: 'preventImage',
        placeholder: 'stackrox/prevent:[current-version]',
        disabled: false
    },
    {
        label: 'Central API Endpoint',
        type: 'text',
        value: 'centralApiEndpoint',
        placeholder: 'central.prevent_net:443',
        disabled: false
    },
    {
        label: 'Namespace',
        type: 'text',
        value: 'namespace',
        disabled: false
    },
    {
        label: 'Image Pull Secret Name',
        type: 'text',
        value: 'imagePullSecret',
        disabled: false
    }
];

export const swarmClusterCreationFormDescriptor = [
    {
        label: 'Name',
        type: 'text',
        value: 'name',
        placeholder: 'Cluster name',
        disabled: false
    },
    {
        label: 'Image name (Prevent location)',
        type: 'text',
        value: 'preventImage',
        placeholder: 'stackrox/prevent:[current-version]',
        disabled: false
    },
    {
        label: 'Central API Endpoint',
        type: 'text',
        value: 'centralApiEndpoint',
        placeholder: 'central.prevent_net:443',
        disabled: false
    },
    {
        label: 'Namespace',
        type: 'text',
        value: 'namespace',
        disabled: false
    },
    {
        label: 'Image Pull Secret Name',
        type: 'text',
        value: 'imagePullSecret',
        disabled: false
    },
    {
        label: 'Disable Swarm TLS',
        type: 'checkbox',
        value: 'disableSwarmTls',
        disabled: false
    }
];
