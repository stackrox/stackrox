import {
    lifecycleStageLabels,
    portExposureLabels,
    envVarSrcLabels,
    rbacPermissionLabels
} from 'messages/common';
import { clientOnlyWhitelistFieldNames } from './whitelistFieldNames';

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
    name: label,
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
    default: false,
    canNegate: false
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
    name: label,
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

// A descriptor for every option on the policy creation page.
const policyStatusDescriptor = [
    {
        label: '',
        header: true,
        jsonpath: 'disabled',
        type: 'toggle',
        required: false,
        reverse: true,
        default: true
    }
];

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
        label: 'Lifecycle Stages',
        jsonpath: 'lifecycleStages',
        type: 'multiselect',
        options: Object.keys(lifecycleStageLabels).map(key => ({
            label: lifecycleStageLabels[key],
            value: key
        })),
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
        label: 'Categories',
        jsonpath: 'categories',
        type: 'multiselect-creatable',
        options: [],
        required: true,
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
        label: 'Restrict to Scope',
        jsonpath: 'scope',
        type: 'scope',
        options: [],
        required: false,
        default: true
    },
    {
        label: 'Whitelist by Scope',
        jsonpath: clientOnlyWhitelistFieldNames.WHITELISTED_DEPLOYMENT_SCOPES,
        type: 'whitelistScope',
        options: [],
        required: false,
        default: true
    },
    {
        label: 'Images Whitelist (Build Lifecycle only)',
        jsonpath: clientOnlyWhitelistFieldNames.WHITELISTED_IMAGE_NAMES,
        type: 'multiselect-creatable',
        options: [],
        required: false,
        default: true
    }
];

const policyConfigurationDescriptor = [
    {
        label: 'Image Registry',
        name: 'Image Registry',
        jsonpath: 'fields.imageName.registry',
        type: 'text',
        placeholder: 'docker.io',
        required: false,
        default: false,
        canNegate: true
        // canBooleanLogic:
    },
    {
        label: 'Image Remote',
        name: 'Image Remote',
        jsonpath: 'fields.imageName.remote',
        type: 'text',
        placeholder: 'library/nginx',
        required: false,
        default: false,
        canNegate: false
    },
    {
        label: 'Image Tag',
        name: 'Image Tag',
        jsonpath: 'fields.imageName.tag',
        type: 'text',
        placeholder: 'latest',
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Days since image was created',
        // does not map to options -- using short name
        name: 'Image Age',
        jsonpath: 'fields.imageAgeDays',
        type: 'number',
        placeholder: '1',
        required: false,
        default: false,
        canNegate: false
    },
    {
        label: 'Days since image was last scanned',
        // does not map to options -- using short name
        name: 'Image Scan Age',
        jsonpath: 'fields.scanAgeDays',
        type: 'number',
        placeholder: '1',
        required: false,
        default: false,
        canNegate: false
    },
    {
        label: 'Dockerfile Line',
        // does not map to options -- using field name in doc
        name: 'Dockerfile Line',
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
        default: false,
        canNegate: false
    },
    {
        label: 'Image is NOT Scanned',
        // using short name
        name: 'Unscanned Image',
        jsonpath: 'fields.noScanExists',
        type: 'toggle',
        required: false,
        default: false,
        defaultValue: true,
        disabled: true,
        canNegate: false
    },
    {
        label: 'CVSS',
        name: 'CVSS',
        jsonpath: 'fields.cvss',
        type: 'group',
        jsonpaths: [
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
        default: false,
        canNegate: false
    },
    {
        label: 'Fixed By',
        // does not map to options -- using field name in doc
        name: 'Fixed By',
        jsonpath: 'fields.fixedBy',
        type: 'text',
        placeholder: '.*',
        required: false,
        default: false,
        canNegate: false
    },
    {
        label: 'CVE',
        name: 'CVE',
        jsonpath: 'fields.cve',
        type: 'text',
        placeholder: 'CVE-2017-11882',
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Image Component',
        // does not map to options -- using field name in doc
        name: 'Image Component',
        jsonpath: 'fields.component',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'fields.component.name',
                type: 'text',
                placeholder: 'example'
            },
            {
                jsonpath: 'fields.component.version',
                type: 'text',
                placeholder: '1.2.[0-9]+'
            }
        ],
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Environment Variable',
        // does not map to options -- using field name in doc
        name: 'Environment Variable',
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
            },
            {
                jsonpath: 'fields.env.envVarSource',
                type: 'select',
                options: Object.keys(envVarSrcLabels).map(key => ({
                    label: envVarSrcLabels[key],
                    value: key
                })),
                placeholder: 'Value From'
            }
        ],
        required: false,
        default: false,
        canNegate: false
    },
    {
        label: 'Disallowed Annotation',
        // does not map to options -- using field name in doc
        name: 'Disallowed Annotation',
        jsonpath: 'fields.disallowedAnnotation',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'fields.disallowedAnnotation.key',
                type: 'text',
                placeholder: 'admission.stackrox.io/break-glass'
            },
            {
                jsonpath: 'fields.disallowedAnnotation.value',
                type: 'text',
                placeholder: ''
            }
        ],
        required: false,
        default: false,
        canNegate: false
    },
    {
        label: 'Required Label',
        // does not map to options -- using field name in doc
        name: 'Required Label',
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
        default: false,
        canNegate: false
    },
    {
        label: 'Required Annotation',
        // does not map to options -- using field name in doc
        name: 'Required Annotation',
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
        default: false,
        canNegate: false
    },
    {
        label: 'Volume Name',
        name: 'Volume Name',
        jsonpath: 'fields.volumePolicy.name',
        type: 'text',
        placeholder: 'docker-socket',
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Volume Source',
        name: 'Volume Source',
        jsonpath: 'fields.volumePolicy.source',
        type: 'text',
        placeholder: '/var/run/docker.sock',
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Volume Destination',
        name: 'Volume Destination',
        jsonpath: 'fields.volumePolicy.destination',
        type: 'text',
        placeholder: '/var/run/docker.sock',
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Volume Type',
        name: 'Volume Type',
        jsonpath: 'fields.volumePolicy.type',
        type: 'text',
        placeholder: 'bind, secret',
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Writable Volume',
        // does not map to options -- using field name in doc
        name: 'Writable Volume',
        jsonpath: 'fields.volumePolicy.readOnly',
        type: 'toggle',
        required: false,
        default: false,
        defaultValue: false,
        reverse: true,
        canNegate: false
    },
    {
        label: 'Protocol',
        name: 'Exposed Port Protocol',
        jsonpath: 'fields.portPolicy.protocol',
        type: 'text',
        placeholder: 'tcp',
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Port',
        name: 'Exposed Port',
        jsonpath: 'fields.portPolicy.port',
        type: 'number',
        placeholder: '22',
        required: false,
        default: false,
        canNegate: true
    },
    cpuResource('Container CPU Request', 'containerResourcePolicy', 'cpuResourceRequest'),
    cpuResource('Container CPU Limit', 'containerResourcePolicy', 'cpuResourceLimit'),
    memoryResource('Container Memory Request', 'containerResourcePolicy', 'memoryResourceRequest'),
    memoryResource('Container Memory Limit', 'containerResourcePolicy', 'memoryResourceLimit'),
    {
        label: 'Privileged',
        name: 'Privileged Container',
        jsonpath: 'fields.privileged',
        type: 'toggle',
        required: false,
        default: false,
        defaultValue: true,
        disabled: true,
        canNegate: false
    },
    {
        label: 'Read-Only Root Filesystem',
        name: 'Read-Only Root Filesystem',
        jsonpath: 'fields.readOnlyRootFs',
        type: 'toggle',
        required: false,
        default: false,
        defaultValue: false,
        disabled: true,
        canNegate: false
    },
    {
        label: 'Drop Capabilities',
        name: 'Drop Capabilities',
        jsonpath: 'fields.dropCapabilities',
        type: 'multiselect',
        options: [...capabilities],
        required: false,
        default: false,
        canNegate: false
    },
    {
        label: 'Add Capabilities',
        name: 'Add Capabilities',
        jsonpath: 'fields.addCapabilities',
        type: 'multiselect',
        options: [...capabilities],
        required: false,
        default: false,
        canNegate: false
    },
    {
        label: 'Process Name',
        name: 'Process Name',
        jsonpath: 'fields.processPolicy.name',
        type: 'text',
        placeholder: 'apt-get',
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Process Ancestor',
        name: 'Process Ancestor',
        jsonpath: 'fields.processPolicy.ancestor',
        type: 'text',
        placeholder: 'java',
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Process Args',
        name: 'Process Arguments',
        jsonpath: 'fields.processPolicy.args',
        type: 'text',
        placeholder: 'install nmap',
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Process UID',
        name: 'Process UID',
        jsonpath: 'fields.processPolicy.uid',
        type: 'text',
        placeholder: '0',
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Port Exposure',
        name: 'Port Exposure Method',
        jsonpath: 'fields.portExposurePolicy.exposureLevels',
        type: 'multiselect',
        options: Object.keys(portExposureLabels)
            .filter(key => key !== 'INTERNAL')
            .map(key => ({
                label: portExposureLabels[key],
                value: key
            })),
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Writable Host Mount',
        name: 'Writable Host Mount',
        jsonpath: 'fields.hostMountPolicy.readOnly',
        type: 'toggle',
        required: false,
        default: false,
        defaultValue: false,
        reverse: true,
        disabled: true,
        canNegate: false
    },
    {
        label: 'Whitelists Enabled',
        name: 'Unexpected Process Executed',
        jsonpath: 'fields.whitelistEnabled',
        type: 'toggle',
        required: false,
        default: false,
        defaultValue: false,
        reverse: false,
        canNegate: false
    },
    {
        label: 'Minimum RBAC Permissions',
        name: 'Minimum RBAC Permissions',
        jsonpath: 'fields.permissionPolicy.permissionLevel',
        type: 'select',
        options: Object.keys(rbacPermissionLabels).map(key => ({
            label: rbacPermissionLabels[key],
            value: key
        })),
        required: false,
        default: false,
        canNegate: true
    },
    {
        label: 'Required Image Label',
        name: 'Required Image Label',
        jsonpath: 'fields.requiredImageLabel',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'fields.requiredImageLabel.key',
                type: 'text',
                placeholder: 'requiredLabelKey.*'
            },
            {
                jsonpath: 'fields.requiredImageLabel.value',
                type: 'text',
                placeholder: 'requiredValue.*'
            }
        ],
        required: false,
        default: false,
        canNegate: false
    },
    {
        label: 'Disallowed Image Label',
        name: 'Disallowed Image Label',
        jsonpath: 'fields.disallowedImageLabel',
        type: 'group',
        jsonpaths: [
            {
                jsonpath: 'fields.disallowedImageLabel.key',
                type: 'text',
                placeholder: 'disallowedLabelKey.*'
            },
            {
                jsonpath: 'fields.disallowedImageLabel.value',
                type: 'text',
                placeholder: 'disallowedValue.*'
            }
        ],
        required: false,
        default: false,
        canNegate: false
    }
];

export const policyStatus = {
    header: 'Enable Policy',
    descriptor: policyStatusDescriptor,
    dataTestId: 'policyStatusField'
};

export const policyDetails = {
    header: 'Policy Summary',
    descriptor: policyDetailsFormDescriptor,
    dataTestId: 'policyDetailsFields'
};

export const policyConfiguration = {
    header: 'Policy Criteria',
    descriptor: policyConfigurationDescriptor,
    dataTestId: 'policyConfigurationFields'
};
