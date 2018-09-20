const equalityOptions = [
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
];

const cpuResource = (label, policy, field) => ({
    label,
    jsonpath: `fields.${policy}.${field}`,
    type: 'group',
    jsonpaths: [
        {
            jsonpath: `fields.${policy}.${field}.op`,
            type: 'select',
            options: equalityOptions
        },
        {
            jsonpath: `fields.${policy}.${field}.value`,
            type: 'number',
            placeholder: '# of cores',
            min: 0,
            step: 0.1
        }
    ],
    required: false,
    default: false
});

const capabilities = [
    { label: 'CAP_AUDIT_CONTROL', value: 'CAP_AUDIT_CONTROL' },
    { label: 'CAP_AUDIT_READ', value: 'CAP_AUDIT_READ' },
    { label: 'CAP_AUDIT_WRITE', value: 'CAP_AUDIT_WRITE' },
    { label: 'CAP_BLOCK_SUSPEND', value: 'CAP_BLOCK_SUSPEND' },
    { label: 'CAP_CHOWN', value: 'CAP_CHOWN' },
    { label: 'CAP_DAC_OVERRIDE', value: 'CAP_DAC_OVERRIDE' },
    { label: 'CAP_DAC_READ_SEARCH', value: 'CAP_DAC_READ_SEARCH' },
    { label: 'CAP_FOWNER', value: 'CAP_FOWNER' },
    { label: 'CAP_FSETID', value: 'CAP_FSETID' },
    { label: 'CAP_IPC_LOCK', value: 'CAP_IPC_LOCK' },
    { label: 'CAP_IPC_OWNER', value: 'CAP_IPC_OWNER' },
    { label: 'CAP_KILL', value: 'CAP_KILL' },
    { label: 'CAP_LEASE', value: 'CAP_LEASE' },
    { label: 'CAP_LINUX_IMMUTABLE', value: 'CAP_LINUX_IMMUTABLE' },
    { label: 'CAP_MAC_ADMIN', value: 'CAP_MAC_ADMIN' },
    { label: 'CAP_MAC_OVERRIDE', value: 'CAP_MAC_OVERRIDE' },
    { label: 'CAP_MKNOD', value: 'CAP_MKNOD' },
    { label: 'CAP_NET_ADMIN', value: 'CAP_NET_ADMIN' },
    { label: 'CAP_NET_BIND_SERVICE', value: 'CAP_NET_BIND_SERVICE' },
    { label: 'CAP_NET_BROADCAST', value: 'CAP_NET_BROADCAST' },
    { label: 'CAP_NET_RAW', value: 'CAP_NET_RAW' },
    { label: 'CAP_SETGID', value: 'CAP_SETGID' },
    { label: 'CAP_SETFCAP', value: 'CAP_SETFCAP' },
    { label: 'CAP_SETPCAP', value: 'CAP_SETPCAP' },
    { label: 'CAP_SETUID', value: 'CAP_SETUID' },
    { label: 'CAP_SYS_ADMIN', value: 'CAP_SYS_ADMIN' },
    { label: 'CAP_SYS_BOOT', value: 'CAP_SYS_BOOT' },
    { label: 'CAP_SYS_CHROOT', value: 'CAP_SYS_CHROOT' },
    { label: 'CAP_SYS_MODULE', value: 'CAP_SYS_MODULE' },
    { label: 'CAP_SYS_NICE', value: 'CAP_SYS_NICE' },
    { label: 'CAP_SYS_PACCT', value: 'CAP_SYS_PACCT' },
    { label: 'CAP_SYS_PTRACE', value: 'CAP_SYS_PTRACE' },
    { label: 'CAP_SYS_RAWIO', value: 'CAP_SYS_RAWIO' },
    { label: 'CAP_SYS_RESOURCE', value: 'CAP_SYS_RESOURCE' },
    { label: 'CAP_SYS_TIME', value: 'CAP_SYS_TIME' },
    { label: 'CAP_SYS_TTY_CONFIG', value: 'CAP_SYS_TTY_CONFIG' },
    { label: 'CAP_SYSLOG', value: 'CAP_SYSLOG' },
    { label: 'CAP_WAKE_ALARM', value: 'CAP_WAKE_ALARM' }
];

const memoryResource = (label, policy, field) => ({
    label,
    jsonpath: `fields.${policy}.${field}`,
    type: 'group',
    jsonpaths: [
        {
            jsonpath: `fields.${policy}.${field}.op`,
            type: 'select',
            options: equalityOptions
        },
        {
            jsonpath: `fields.${policy}.${field}.value`,
            type: 'number',
            placeholder: '# MB',
            min: 0
        }
    ],
    required: false,
    default: false
});

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

const policyConfigurationDescriptor = [
    {
        label: 'Image Registry',
        jsonpath: 'fields.imageName.registry',
        type: 'text',
        placeholder: 'docker.io',
        required: false,
        default: false
    },
    {
        label: 'Image Remote',
        jsonpath: 'fields.imageName.remote',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'Image Tag',
        jsonpath: 'fields.imageName.tag',
        type: 'text',
        placeholder: 'latest',
        required: false,
        default: false
    },
    {
        label: 'Days since Image created',
        jsonpath: 'fields.imageAgeDays',
        type: 'number',
        placeholder: '1 Day Ago',
        required: false,
        default: false
    },
    {
        label: 'Days since Image scanned',
        jsonpath: 'fields.scanAgeDays',
        type: 'number',
        placeholder: '1 Day Ago',
        required: false,
        default: false
    },
    {
        label: 'Dockerfile Line',
        jsonpath: 'fields.lineRule',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'fields.lineRule.instruction',
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
                jsonpath: 'fields.lineRule.value',
                type: 'text',
                placeholder: '.*example.*'
            }
        ],
        required: false,
        default: false
    },
    {
        label: 'Image is NOT Scanned',
        jsonpath: 'fields.noScanExists',
        type: 'select',
        options: [{ label: 'True', value: true }],
        required: false,
        default: false
    },
    {
        label: 'CVSS',
        jsonpath: 'fields.cvss',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'fields.cvss.mathOp',
                type: 'select',
                options: [
                    { label: 'Max score', value: 'MAX' },
                    { label: 'Average score', value: 'AVG' },
                    { label: 'Min score', value: 'MIN' }
                ]
            },
            {
                jsonpath: 'fields.cvss.op',
                type: 'select',
                options: equalityOptions
            },
            {
                jsonpath: 'fields.cvss.value',
                type: 'number',
                placeholder: '0-10',
                max: 10,
                min: 0,
                step: 0.1
            }
        ],
        required: false,
        default: false
    },
    {
        label: 'CVE',
        jsonpath: 'fields.cve',
        type: 'text',
        placeholder: 'CVE-2017-11882',
        required: false,
        default: false
    },
    {
        label: 'Component',
        jsonpath: 'fields.component',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'fields.component.name',
                type: 'text',
                placeholder: '^example*'
            },
            {
                jsonpath: 'fields.component.version',
                type: 'text',
                placeholder: '^v1.2.0$'
            }
        ],
        required: false,
        default: false
    },
    {
        label: 'Environment',
        jsonpath: 'fields.env',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'fields.env.key',
                type: 'text',
                placeholder: 'Key'
            },
            {
                jsonpath: 'fields.env.value',
                type: 'text',
                placeholder: 'Value'
            }
        ],
        required: false,
        default: false
    },
    {
        label: 'Required Label',
        jsonpath: 'fields.requiredLabel',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'fields.requiredLabel.key',
                type: 'text',
                placeholder: 'owner'
            },
            {
                jsonpath: 'fields.requiredLabel.value',
                type: 'text',
                placeholder: '.*'
            }
        ],
        required: false,
        default: false
    },
    {
        label: 'Required Annotation',
        jsonpath: 'fields.requiredAnnotation',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'fields.requiredAnnotation.key',
                type: 'text',
                placeholder: 'owner'
            },
            {
                jsonpath: 'fields.requiredAnnotation.value',
                type: 'text',
                placeholder: '.*'
            }
        ],
        required: false,
        default: false
    },
    {
        label: 'Command',
        jsonpath: 'fields.command',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'Arguments',
        jsonpath: 'fields.args',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'Directory',
        jsonpath: 'fields.directory',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'User',
        jsonpath: 'fields.user',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'Volume Name',
        jsonpath: 'fields.volumePolicy.name',
        type: 'text',
        placeholder: '/var/run/docker.sock',
        required: false,
        default: false
    },
    {
        label: 'Volume Source',
        jsonpath: 'configurationPolicy.volumePolicy.source',
        type: 'text',
        placeholder: '^/var/run/docker.sock$',
        required: false,
        default: false
    },
    {
        label: 'Volume Destination',
        jsonpath: 'configurationPolicy.volumePolicy.destination',
        type: 'text',
        placeholder: '^/var/run/docker.sock$',
        required: false,
        default: false
    },
    {
        label: 'Volume Type',
        jsonpath: 'fields.volumePolicy.type',
        type: 'text',
        placeholder: 'bind, secret',
        required: false,
        default: false
    },
    {
        label: 'Protocol',
        jsonpath: 'fields.portPolicy.protocol',
        type: 'text',
        required: false,
        default: false
    },
    {
        label: 'Port',
        jsonpath: 'fields.portPolicy.port',
        type: 'number',
        required: false,
        default: false
    },
    cpuResource('Container CPU Request', 'containerResourcePolicy', 'cpuResourceRequest'),
    cpuResource('Container CPU Limit', 'containerResourcePolicy', 'cpuResourceLimit'),
    memoryResource('Container Memory Request', 'containerResourcePolicy', 'memoryResourceRequest'),
    memoryResource('Container Memory Limit', 'containerResourcePolicy', 'memoryResourceLimit'),

    cpuResource('Total CPU Request', 'totalResourcePolicy', 'cpuResourceRequest'),
    cpuResource('Total CPU Limit', 'totalResourcePolicy', 'cpuResourceLimit'),
    memoryResource('Total Memory Request', 'totalResourcePolicy', 'memoryResourceRequest'),
    memoryResource('Total Memory Limit', 'totalResourcePolicy', 'memoryResourceLimit'),
    {
        label: 'Privileged',
        jsonpath: 'fields.privileged',
        type: 'select',
        options: [{ label: 'Yes', value: true }, { label: 'No', value: false }],
        required: false,
        default: false
    },
    {
        label: 'Drop Capabilities',
        jsonpath: 'fields.dropCapabilities',
        type: 'multiselect',
        options: [...capabilities],
        required: false,
        default: false
    },
    {
        label: 'Add Capabilities',
        jsonpath: 'fields.addCapabilities',
        type: 'multiselect',
        options: [...capabilities],
        required: false,
        default: false
    }
];

const policyFormFields = {
    policyDetails: {
        header: 'Policy Details',
        descriptor: policyDetailsFormDescriptor
    },
    policyConfiguration: {
        header: 'Policy Configuration',
        descriptor: policyConfigurationDescriptor
    }
};

export default policyFormFields;
