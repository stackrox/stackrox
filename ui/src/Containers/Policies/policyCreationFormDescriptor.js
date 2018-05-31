const policyDetailsFormDescriptor = [
    {
        label: 'Name',
        jsonpath: 'name',
        type: 'text',
        required: true,
        default: true
    },
    {
        label: 'Severity',
        jsonpath: 'severity',
        type: 'select',
        options: [
            { label: 'Critical', value: 'CRITICAL_SEVERITY' },
            { label: 'High', value: 'HIGH_SEVERITY' },
            { label: 'Medium', value: 'MEDIUM_SEVERITY' },
            { label: 'Low', value: 'LOW_SEVERITY' }
        ],
        placeholder: 'Select a severity level',
        required: true,
        default: true
    },
    {
        label: 'Description',
        jsonpath: 'description',
        type: 'textarea',
        placeholder: 'What does this policy do?',
        required: false,
        default: true
    },
    {
        label: 'Rationale',
        jsonpath: 'rationale',
        type: 'textarea',
        placeholder: 'Why does this policy exist?',
        required: false,
        default: true
    },
    {
        label: 'Remediation',
        jsonpath: 'remediation',
        type: 'textarea',
        placeholder: 'What can an operator do to resolve any violations?',
        required: false,
        default: true
    },
    {
        label: 'Enable',
        jsonpath: 'disabled',
        exclude: false,
        type: 'select',
        options: [{ label: 'Yes', value: false }, { label: 'No', value: true }],
        required: false,
        default: true
    },
    {
        label: 'Categories',
        jsonpath: 'categories',
        type: 'multiselect-creatable',
        options: [],
        required: true,
        default: true
    },
    {
        label: 'Enforcement Action',
        jsonpath: 'enforcement',
        type: 'select',
        options: [
            { label: 'None', value: 'UNSET_ENFORCEMENT' },
            { label: 'Scale to Zero Replicas', value: 'SCALE_TO_ZERO_ENFORCEMENT' },
            {
                label: 'Add an Unsatisfiable Node Constraint',
                value: 'UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT'
            }
        ],
        required: false,
        default: true
    },
    {
        label: 'Notifications',
        jsonpath: 'notifiers',
        type: 'multiselect',
        options: [],
        required: false,
        default: true
    },
    {
        label: 'Restrict to Clusters',
        jsonpath: 'scope',
        type: 'multiselect',
        options: [],
        required: false,
        default: true
    },
    {
        label: 'Deployments Whitelist',
        jsonpath: 'deployments',
        type: 'multiselect',
        options: [],
        required: false,
        default: true
    }
];

const imagePolicyFormDescriptor = [
    {
        label: 'Image Registry',
        jsonpath: 'imagePolicy.imageName.registry',
        type: 'text',
        placeholder: 'docker.io',
        required: false,
        default: false
    },
    {
        label: 'Image Namespace',
        jsonpath: 'imagePolicy.imageName.namespace',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'Image Repository',
        jsonpath: 'imagePolicy.imageName.repo',
        type: 'text',
        placeholder: 'nginx',
        required: false,
        default: false
    },
    {
        label: 'Image Tag',
        jsonpath: 'imagePolicy.imageName.tag',
        type: 'text',
        placeholder: 'latest',
        required: false,
        default: false
    },
    {
        label: 'Days since Image created',
        jsonpath: 'imagePolicy.imageAgeDays',
        type: 'number',
        placeholder: '1 Day Ago',
        min: 1,
        max: Number.MAX_SAFE_INTEGER,
        required: false,
        default: false
    },
    {
        label: 'Days since Image scanned',
        jsonpath: 'imagePolicy.scanAgeDays',
        type: 'number',
        placeholder: '1 Day Ago',
        min: 1,
        max: Number.MAX_SAFE_INTEGER,
        required: false,
        default: false
    },
    {
        label: 'Dockerfile Line',
        jsonpath: 'imagePolicy.lineRule',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'imagePolicy.lineRule.instruction',
                type: 'select',
                options: [
                    { label: 'FROM', value: 'FROM' },
                    { label: 'LABEL', value: 'LABEL' },
                    { label: 'RUN', value: 'RUN' },
                    { label: 'CMD', value: 'CMD' },
                    { label: 'EXPOSE', value: 'EXPOSE' },
                    { label: 'ENV', value: 'ENV' },
                    { label: 'ADD', value: 'ADD' },
                    { label: 'COPY', value: 'COPY' },
                    { label: 'ENTRYPOINT', value: 'ENTRYPOINT' },
                    { label: 'VOLUME', value: 'VOLUME' },
                    { label: 'USER', value: 'USER' },
                    { label: 'WORKDIR', value: 'WORKDIR' },
                    { label: 'ONBUILD', value: 'ONBUILD' }
                ]
            },
            {
                jsonpath: 'imagePolicy.lineRule.value',
                type: 'text',
                placeholder: '.*example.*'
            }
        ],
        required: false,
        default: false
    },
    {
        label: 'Image is NOT Scanned',
        jsonpath: 'imagePolicy.scanExists',
        type: 'select',
        options: [{ label: 'True', value: true }],
        required: false,
        default: false
    },
    {
        label: 'CVSS',
        jsonpath: 'imagePolicy.cvss',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'imagePolicy.cvss.mathOp',
                type: 'select',
                options: [
                    { label: 'Max score', value: 'MAX' },
                    { label: 'Average score', value: 'AVG' },
                    { label: 'Min score', value: 'MIN' }
                ]
            },
            {
                jsonpath: 'imagePolicy.cvss.op',
                type: 'select',
                options: [
                    { label: 'Is greater than', value: 'GREATER_THAN' },
                    {
                        label: 'Is greater than or equal to',
                        value: 'GREATER_THAN_OR_EQUALS'
                    },
                    { label: 'Is equal to', value: 'EQUALS' },
                    {
                        label: 'Is less than or equal to',
                        value: 'LESS_THAN_OR_EQUALS'
                    },
                    { label: 'Is less than', value: 'LESS_THAN' }
                ]
            },
            {
                jsonpath: 'imagePolicy.cvss.value',
                type: 'number',
                placeholder: '0-10',
                max: 10,
                min: 0
            }
        ],
        required: false,
        default: false
    },
    {
        label: 'CVE',
        jsonpath: 'imagePolicy.cve',
        type: 'text',
        placeholder: 'CVE-2017-11882',
        required: false,
        default: false
    },
    {
        label: 'Component',
        jsonpath: 'imagePolicy.component',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'imagePolicy.component.name',
                type: 'text',
                placeholder: '^example*'
            },
            {
                jsonpath: 'imagePolicy.component.version',
                type: 'text',
                placeholder: '^v1.2.0$'
            }
        ],
        required: false,
        default: false
    }
];

const configurationPolicyFormDescriptor = [
    {
        label: 'Environment',
        jsonpath: 'configurationPolicy.env',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'configurationPolicy.env.key',
                type: 'text',
                placeholder: 'Key'
            },
            {
                jsonpath: 'configurationPolicy.env.value',
                type: 'text',
                placeholder: 'Value'
            }
        ],
        required: false,
        default: false
    },
    {
        label: 'Required Label',
        jsonpath: 'configurationPolicy.requiredLabel',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'configurationPolicy.requiredLabel.key',
                type: 'text',
                placeholder: 'owner'
            },
            {
                jsonpath: 'configurationPolicy.requiredLabel.value',
                type: 'text',
                placeholder: '.*'
            }
        ],
        required: false,
        default: false
    },
    {
        label: 'Required Annotation',
        jsonpath: 'configurationPolicy.requiredAnnotation',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'configurationPolicy.requiredAnnotation.key',
                type: 'text',
                placeholder: 'owner'
            },
            {
                jsonpath: 'configurationPolicy.requiredAnnotation.value',
                type: 'text',
                placeholder: '.*'
            }
        ],
        required: false,
        default: false
    },
    {
        label: 'Command',
        jsonpath: 'configurationPolicy.command',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'Arguments',
        jsonpath: 'configurationPolicy.args',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'Directory',
        jsonpath: 'configurationPolicy.directory',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'User',
        jsonpath: 'configurationPolicy.user',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'Volume Name',
        jsonpath: 'configurationPolicy.volumePolicy.name',
        type: 'text',
        placeholder: '/var/run/docker.sock',
        required: false,
        default: false
    },
    {
        label: 'Volume Path',
        jsonpath: 'configurationPolicy.volumePolicy.path',
        type: 'text',
        placeholder: '^/var/run/docker.sock$',
        required: false,
        default: false
    },
    {
        label: 'Volume Type',
        jsonpath: 'configurationPolicy.volumePolicy.type',
        type: 'text',
        placeholder: 'bind, secret',
        required: false,
        default: false
    },
    {
        label: 'Protocol',
        jsonpath: 'configurationPolicy.portPolicy.protocol',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'Port',
        jsonpath: 'configurationPolicy.portPolicy.port',
        type: 'number',
        required: false,
        default: false
    }
];

const privilegePolicyFormDescriptor = [
    {
        label: 'Privileged',
        jsonpath: 'privilegePolicy.privileged',
        type: 'select',
        options: [{ label: 'Yes', value: true }, { label: 'No', value: false }],
        required: false,
        default: false
    },
    {
        label: 'Drop Capabilities',
        jsonpath: 'privilegePolicy.dropCapabilities',
        type: 'multiselect',
        options: [],
        required: false,
        default: false
    },
    {
        label: 'Add Capabilities',
        jsonpath: 'privilegePolicy.addCapabilities',
        type: 'multiselect',
        options: [],
        required: false,
        default: false
    }
];

const policyFormFields = {
    policyDetails: {
        header: 'Policy Details',
        descriptor: policyDetailsFormDescriptor
    },
    imagePolicy: {
        header: 'Image Assurance',
        descriptor: imagePolicyFormDescriptor
    },
    configurationPolicy: {
        header: 'Container Configuration',
        descriptor: configurationPolicyFormDescriptor
    },
    privilegePolicy: {
        header: 'Privileges and Capabilities',
        descriptor: privilegePolicyFormDescriptor
    }
};

export default policyFormFields;
