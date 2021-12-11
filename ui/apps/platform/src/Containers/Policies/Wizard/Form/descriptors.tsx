import {
    portExposureLabels,
    envVarSrcLabels,
    rbacPermissionLabels,
    policyCriteriaCategories,
    mountPropagationLabels,
    seccompProfileTypeLabels,
    severityRatings,
} from 'messages/common';

const equalityOptions = [
    { label: 'Is greater than', value: '>' },
    {
        label: 'Is greater than or equal to',
        value: '>=',
    },
    { label: 'Is equal to', value: '=' },
    {
        label: 'Is less than or equal to',
        value: '<=',
    },
    { label: 'Is less than', value: '<' },
];

const cpuResource = (label) => ({
    label,
    name: label,
    category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
    type: 'group',
    subComponents: [
        {
            type: 'select',
            options: equalityOptions,
            subpath: 'key',
        },
        {
            type: 'number',
            placeholder: '# of cores',
            min: 0,
            step: 0.1,
            subpath: 'value',
        },
    ],
    canBooleanLogic: true,
});

const capabilities = [
    'AUDIT_CONTROL',
    'AUDIT_READ',
    'AUDIT_WRITE',
    'BLOCK_SUSPEND',
    'CHOWN',
    'DAC_OVERRIDE',
    'DAC_READ_SEARCH',
    'FOWNER',
    'FSETID',
    'IPC_LOCK',
    'IPC_OWNER',
    'KILL',
    'LEASE',
    'LINUX_IMMUTABLE',
    'MAC_ADMIN',
    'MAC_OVERRIDE',
    'MKNOD',
    'NET_ADMIN',
    'NET_BIND_SERVICE',
    'NET_BROADCAST',
    'NET_RAW',
    'SETGID',
    'SETFCAP',
    'SETPCAP',
    'SETUID',
    'SYS_ADMIN',
    'SYS_BOOT',
    'SYS_CHROOT',
    'SYS_MODULE',
    'SYS_NICE',
    'SYS_PACCT',
    'SYS_PTRACE',
    'SYS_RAWIO',
    'SYS_RESOURCE',
    'SYS_TIME',
    'SYS_TTY_CONFIG',
    'SYSLOG',
    'WAKE_ALARM',
].map((cap) => ({ label: cap, value: cap }));

const APIVerbs = ['CREATE', 'DELETE', 'GET', 'PATCH', 'UPDATE'].map((verb) => ({
    label: verb,
    value: verb,
}));

const memoryResource = (label) => ({
    label,
    name: label,
    category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
    type: 'group',
    subComponents: [
        {
            type: 'select',
            options: equalityOptions,
            subpath: 'key',
        },
        {
            type: 'number',
            placeholder: '# MB',
            min: 0,
            subpath: 'value',
        },
    ],
    canBooleanLogic: true,
});

// A form descriptor for every option (key) on the policy criteria form page.
/* 
    e.g. 
    {
        label: 'Image Tag',
        name: 'Image Tag',
        negatedName: `Image tag doesn't match`,
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'latest',
        canBooleanLogic: true,
    },

    label: for legacy policy alert labels 
    name: the string used to display UI and send to backend
    negatedName: string used to display UI when negated 
        (if this does not exist, the UI assumes that the field cannot be negated)
    longName: string displayed in the UI in the Policy Field Card (not in draggable key)
    category: the category grouping for the policy criteria (collapsible group in keys)
    type: the type of form field to render when dragged to the Policy Field Card
    subComponents: subfields the field renders when dragged to Policy Field Card if 'group' type
    radioButtons: button options if 'radio' type
    options: options if 'select' or 'multiselect' or 'multiselect-creatable' type
    placeholder: string to display as placeholder if applicable
    canBooleanLogic: indicates whether the field supports the AND/OR boolean operator
        (UI assumes false by default)
    defaultValue: the default value to set, if provided
    disabled: disables the field entirely
    reverse: will reverse boolean value on store 
 */

type SubComponent = {
    type: string;
    options?: {
        label: string;
        value: string;
    }[];
    subpath: string;
    placeholder?: string;
    label?: string;
    min?: number;
    max?: number;
    step?: number;
};

export type Descriptor = {
    label: string;
    name: string;
    longName?: string;
    shortName?: string;
    negatedName?: string;
    category: string;
    type: string;
    subComponents?: SubComponent[];
    radioButtons?: { text: string; value: string | boolean }[];
    options?: { label: string; value: string }[];
    placeholder?: string;
    canBooleanLogic?: boolean;
    default?: boolean;
    defaultValue?: string | boolean;
    disabled?: boolean;
    reverse?: boolean;
};

export const policyConfigurationDescriptor: Descriptor[] = [
    {
        label: 'Image Registry',
        name: 'Image Registry',
        longName: 'Image pulled from registry',
        negatedName: 'Image not pulled from registry',
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'docker.io',
        canBooleanLogic: true,
    },
    {
        label: 'Image Remote',
        name: 'Image Remote',
        longName: 'Image name in the registry',
        negatedName: `Image name in the registry doesn't match`,
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'library/nginx',
        canBooleanLogic: true,
    },
    {
        label: 'Image Tag',
        name: 'Image Tag',
        negatedName: `Image tag doesn't match`,
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'latest',
        canBooleanLogic: true,
    },
    {
        label: 'Days since image was created',
        name: 'Image Age',
        longName: 'Minimum days since image was built',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'number',
        placeholder: '1',
        canBooleanLogic: false,
    },
    {
        label: 'Days since image was last scanned',
        name: 'Image Scan Age',
        longName: 'Minimum days since last image scan',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'number',
        placeholder: '1',
        canBooleanLogic: false,
    },
    {
        label: 'Image User',
        name: 'Image User',
        negatedName: `Image user is not`,
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'text',
        placeholder: '0',
        canBooleanLogic: false,
    },
    {
        label: 'Dockerfile Line',
        name: 'Dockerfile Line',
        longName: 'Disallowed Dockerfile line',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'group',
        subComponents: [
            {
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
                    { label: 'ONBUILD', value: 'ONBUILD' },
                ],
                label: 'Instruction',
                subpath: 'key',
            },
            {
                type: 'text',
                label: 'Arguments',
                placeholder: 'Any',
                subpath: 'value',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Image is NOT Scanned',
        name: 'Unscanned Image',
        longName: 'Image Scan Status',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Scanned',
                value: false,
            },
            {
                text: 'Not scanned',
                value: true,
            },
        ],
        defaultValue: true,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        label: 'CVSS',
        name: 'CVSS',
        longName: 'Common Vulnerability Scoring System (CVSS) Score',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'group',
        subComponents: [
            {
                type: 'select',
                options: equalityOptions,
                subpath: 'key',
            },
            {
                type: 'number',
                placeholder: '0-10',
                max: 10,
                min: 0,
                step: 0.1,
                subpath: 'value',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Severity',
        name: 'Severity',
        longName: 'Vulnerability Severity Rating',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'group',
        subComponents: [
            {
                type: 'select',
                options: equalityOptions,
                subpath: 'key',
            },
            {
                type: 'select',
                options: Object.keys(severityRatings).map((key) => ({
                    label: severityRatings[key],
                    value: key,
                })),
                subpath: 'value',
            },
        ],
        default: false,
        canBooleanLogic: true,
    },
    {
        label: 'Fixed By',
        name: 'Fixed By',
        longName: 'Version in which vulnerability is fixed',
        negatedName: `Version in which vulnerability is fixed doesn't match`,
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'text',
        placeholder: '.*',
        canBooleanLogic: true,
    },
    {
        label: 'CVE',
        name: 'CVE',
        longName: 'Common Vulnerabilities and Exposures (CVE) identifier',
        negatedName: `Common Vulnerabilities and Exposures (CVE) identifier doesn't match`,
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'text',
        placeholder: 'CVE-2017-11882',
        canBooleanLogic: true,
    },
    {
        label: 'Image Component',
        name: 'Image Component',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'group',
        subComponents: [
            {
                type: 'text',
                label: 'Component Name',
                placeholder: 'example',
                subpath: 'key',
            },
            {
                type: 'text',
                label: 'Version',
                placeholder: 'Any',
                subpath: 'value',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Image OS',
        name: 'Image OS',
        longName: 'Image Operating System',
        negatedName: `Image Operating System doesn't match`,
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'text',
        placeholder: 'ubuntu:19.04',
        canBooleanLogic: true,
    },
    {
        label: 'Environment Variable',
        name: 'Environment Variable',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'group',
        subComponents: [
            {
                type: 'text',
                label: 'Key',
                placeholder: 'Any',
                subpath: 'key',
            },
            {
                type: 'text',
                label: 'Value',
                placeholder: 'Any',
                subpath: 'value',
            },
            {
                type: 'select',
                options: Object.keys(envVarSrcLabels).map((key) => ({
                    label: envVarSrcLabels[key],
                    value: key,
                })),
                label: 'Value From',
                placeholder: 'Select one',
                subpath: 'source',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Disallowed Annotation',
        name: 'Disallowed Annotation',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'group',
        subComponents: [
            {
                type: 'text',
                label: 'Key',
                placeholder: 'Any',
                subpath: 'key',
            },
            {
                type: 'text',
                label: 'Value',
                placeholder: 'Any',
                subpath: 'value',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Required Label',
        name: 'Required Label',
        longName: 'Required Deployment Label',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'group',
        subComponents: [
            {
                type: 'text',
                label: 'Key',
                placeholder: 'owner',
                subpath: 'key',
            },
            {
                type: 'text',
                label: 'Value',
                placeholder: '.*',
                subpath: 'value',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Required Annotation',
        name: 'Required Annotation',
        longName: 'Required Deployment Annotation',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'group',
        subComponents: [
            {
                type: 'text',
                label: 'Key',
                placeholder: 'owner',
                subpath: 'key',
            },
            {
                type: 'text',
                label: 'Value',
                placeholder: '.*',
                subpath: 'value',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Runtime Class',
        name: 'Runtime Class',
        negatedName: `Runtime Class doesn't match`,
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'text',
        placeholder: 'kata',
        canBooleanLogic: true,
    },
    {
        label: 'Volume Name',
        name: 'Volume Name',
        negatedName: `Volume name doesn't match`,
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: 'docker-socket',
        canBooleanLogic: true,
    },
    {
        label: 'Volume Source',
        name: 'Volume Source',
        negatedName: `Volume source doesn't match`,
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: '/var/run/docker.sock',
        canBooleanLogic: true,
    },
    {
        label: 'Volume Destination',
        name: 'Volume Destination',
        negatedName: `Volume destination doesn't match`,
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: '/var/run/docker.sock',
        canBooleanLogic: true,
    },
    {
        label: 'Volume Type',
        name: 'Volume Type',
        negatedName: `Volume type doesn't match`,
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: 'bind, secret',
        canBooleanLogic: true,
    },
    {
        label: 'Writable Mounted Volume',
        name: 'Writable Mounted Volume',
        longName: 'Mounted Volume Writability',
        category: policyCriteriaCategories.STORAGE,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Writable',
                value: true,
            },
            {
                text: 'Read-only',
                value: false,
            },
        ],
        defaultValue: false,
        reverse: true,
        canBooleanLogic: false,
    },
    {
        label: 'Mount Propagation',
        name: 'Mount Propagation',
        negatedName: 'Mount Propagation is not',
        category: policyCriteriaCategories.STORAGE,
        type: 'multiselect',
        options: Object.keys(mountPropagationLabels).map((key) => ({
            label: mountPropagationLabels[key],
            value: key,
        })),
        canBooleanLogic: true,
    },
    {
        label: 'Protocol',
        name: 'Exposed Port Protocol',
        negatedName: `Exposed Port Protocol doesn't match`,
        category: policyCriteriaCategories.NETWORKING,
        type: 'text',
        placeholder: 'tcp',
        canBooleanLogic: true,
    },
    {
        label: 'Exposed Node Port',
        name: 'Exposed Node Port',
        negatedName: `Exposed node port doesn't match`,
        category: policyCriteriaCategories.NETWORKING,
        type: 'number',
        placeholder: '22',
        canBooleanLogic: true,
    },
    {
        label: 'Port',
        name: 'Exposed Port',
        negatedName: `Exposed port doesn't match`,
        category: policyCriteriaCategories.NETWORKING,
        type: 'number',
        placeholder: '22',
        canBooleanLogic: true,
    },
    cpuResource('Container CPU Request'),
    cpuResource('Container CPU Limit'),
    memoryResource('Container Memory Request'),
    memoryResource('Container Memory Limit'),
    {
        label: 'Privileged',
        name: 'Privileged Container',
        longName: 'Privileged Container Status',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Privileged Container',
                value: true,
            },
            {
                text: 'Not a Privileged Container',
                value: false,
            },
        ],
        defaultValue: true,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        label: 'Read-Only Root Filesystem',
        name: 'Read-Only Root Filesystem',
        longName: 'Root Filesystem Writability',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Read-Only',
                value: true,
            },
            {
                text: 'Writable',
                value: false,
            },
        ],
        defaultValue: false,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        label: 'Seccomp Profile Type',
        name: 'Seccomp Profile Type',
        negatedName: 'Seccomp Profile Type is not',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'radioGroup',
        radioButtons: Object.keys(seccompProfileTypeLabels).map((key) => ({
            text: seccompProfileTypeLabels[key],
            value: key,
        })),
        canBooleanLogic: false,
    },
    {
        label: 'Share Host Network Namespace',
        name: 'Host Network',
        longName: 'Host Network',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Uses Host Network Namespace',
                value: true,
            },
            {
                text: 'Does Not Use Host Network Namespace',
                value: false,
            },
        ],
        defaultValue: true,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        label: 'Share Host PID Namespace',
        name: 'Host PID',
        longName: 'Host PID',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Uses Host PID namespace',
                value: true,
            },
            {
                text: 'Does Not Use Host PID namespace',
                value: false,
            },
        ],
        defaultValue: true,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        label: 'Share Host IPC Namespace',
        name: 'Host IPC',
        longName: 'Host IPC',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Uses Host IPC namespace',
                value: true,
            },
            {
                text: 'Does Not Use Host IPC namespace',
                value: false,
            },
        ],
        defaultValue: true,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        label: 'Drop Capabilities',
        name: 'Drop Capabilities',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'select',
        options: [...capabilities],
        canBooleanLogic: true,
    },
    {
        label: 'Add Capabilities',
        name: 'Add Capabilities',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'select',
        options: [...capabilities],
        canBooleanLogic: true,
    },
    {
        label: 'Process Name',
        name: 'Process Name',
        negatedName: `Process name doesn't match`,
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'text',
        placeholder: 'apt-get',
        canBooleanLogic: true,
    },
    {
        label: 'Process Ancestor',
        name: 'Process Ancestor',
        negatedName: `Process ancestor doesn't match`,
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'text',
        placeholder: 'java',
        canBooleanLogic: true,
    },
    {
        label: 'Process Args',
        name: 'Process Arguments',
        negatedName: `Process arguments don't match`,
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'text',
        placeholder: 'install nmap',
        canBooleanLogic: true,
    },
    {
        label: 'Process UID',
        name: 'Process UID',
        negatedName: `Process UID doesn't match`,
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'text',
        placeholder: '0',
        canBooleanLogic: true,
    },
    {
        label: 'Port Exposure',
        name: 'Port Exposure Method',
        negatedName: 'Port Exposure Method is not',
        category: policyCriteriaCategories.NETWORKING,
        type: 'select',
        options: Object.keys(portExposureLabels)
            .filter((key) => key !== 'INTERNAL')
            .map((key) => ({
                label: portExposureLabels[key],
                value: key,
            })),
        canBooleanLogic: true,
    },
    {
        label: 'Writable Host Mount',
        name: 'Writable Host Mount',
        longName: 'Host Mount Writability',
        category: policyCriteriaCategories.STORAGE,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Writable',
                value: true,
            },
            {
                text: 'Read-only',
                value: false,
            },
        ],
        defaultValue: false,
        reverse: true,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        label: 'Process Baselining Enabled',
        name: 'Unexpected Process Executed',
        longName: 'Process Baselining Status',
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'radioGroup',
        radioButtons: [
            { text: 'Unexpected Process', value: true },
            { text: 'Expected Process', value: false },
        ],
        defaultValue: false,
        reverse: false,
        canBooleanLogic: false,
    },
    {
        label: 'Service Account',
        name: 'Service Account',
        longName: 'Service Account Name',
        negatedName: `Service Account Name doesn't match`,
        category: policyCriteriaCategories.KUBERNETES_ACCESS,
        type: 'text',
        canBooleanLogic: true,
    },
    {
        label: 'Minimum RBAC Permissions',
        name: 'Minimum RBAC Permissions',
        longName: 'RBAC permission level is at least',
        negatedName: 'RBAC permission level is less than',
        category: policyCriteriaCategories.KUBERNETES_ACCESS,
        type: 'select',
        options: Object.keys(rbacPermissionLabels).map((key) => ({
            label: rbacPermissionLabels[key],
            value: key,
        })),
        canBooleanLogic: false,
    },
    {
        label: 'Required Image Label',
        name: 'Required Image Label',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'group',
        subComponents: [
            {
                type: 'text',
                label: 'Key',
                placeholder: 'requiredLabelKey.*',
                subpath: 'key',
            },
            {
                type: 'text',
                label: 'Value',
                placeholder: 'requiredValue.*',
                subpath: 'value',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Disallowed Image Label',
        name: 'Disallowed Image Label',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'group',
        subComponents: [
            {
                type: 'text',
                label: 'Key',
                placeholder: 'disallowedLabelKey.*',
                subpath: 'key',
            },
            {
                type: 'text',
                label: 'Value',
                placeholder: 'disallowedValue.*',
                subpath: 'value',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Namespace',
        name: 'Namespace',
        longName: 'Namespace',
        negatedName: `Namespace doesn't match`,
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'text',
        placeholder: 'default',
        canBooleanLogic: true,
    },
    {
        label: 'Container Name',
        name: 'Container Name',
        longName: 'Container Name',
        negatedName: `Container name doesn't match`,
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'text',
        placeholder: 'default',
        canBooleanLogic: true,
    },
    {
        label: 'AppArmor Profile',
        name: 'AppArmor Profile',
        longName: 'AppArmor Profile',
        negatedName: `AppArmor Profile doesn't match`,
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'text',
        placeholder: 'default',
        canBooleanLogic: true,
    },
    {
        label: 'Kubernetes Action',
        name: 'Kubernetes Resource',
        longName: 'Kubernetes Action',
        shortName: 'Kubernetes Action',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'select',
        options: [
            {
                label: 'Pod Exec',
                value: 'PODS_EXEC',
            },
            {
                label: 'Pods Port Forward',
                value: 'PODS_PORTFORWARD',
            },
        ],
        canBooleanLogic: false,
    },
    {
        label: 'Kubernetes User Name',
        name: 'Kubernetes User Name',
        negatedName: "Kubernetes User Name doesn't match",
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'Kubernetes User Groups',
        name: 'Kubernetes User Groups',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
];

export const auditLogDescriptor: Descriptor[] = [
    {
        label: 'Kubernetes Resource',
        name: 'Kubernetes Resource',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'select',
        options: [
            {
                label: 'Config Maps',
                value: 'CONFIGMAPS',
            },
            {
                label: 'Secrets',
                value: 'SECRETS',
            },
        ],
        canBooleanLogic: false,
    },
    {
        label: 'Kubernetes API Verb',
        name: 'Kubernetes API Verb',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'select',
        options: APIVerbs,
        canBooleanLogic: false,
    },
    {
        label: 'Kubernetes Resource Name',
        name: 'Kubernetes Resource Name',
        negatedName: "Kubernetes Resource Name doesn't match",
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'Kubernetes User Name',
        name: 'Kubernetes User Name',
        negatedName: "Kubernetes User Name doesn't match",
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'Kubernetes User Group',
        name: 'Kubernetes User Groups',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'User Agent',
        name: 'User Agent',
        negatedName: "User Agent doesn't match",
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'Source IP Address',
        name: 'Source IP Address',
        negatedName: "Source IP Address doesn't match",
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'Is Impersonated User',
        name: 'Is Impersonated User',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'radioGroup',
        radioButtons: [
            { text: 'True', value: true },
            { text: 'False', value: false },
        ],
        canBooleanLogic: false,
    },
];

export const networkDetectionDescriptor: Descriptor[] = [
    {
        label: 'Network Baselining Enabled',
        name: 'Unexpected Network Flow Detected',
        longName: 'Network Baselining Status',
        category: policyCriteriaCategories.NETWORKING,
        type: 'radioGroup',
        radioButtons: [
            { text: 'Unexpected Network Flow', value: true },
            { text: 'Expected Network Flow', value: false },
        ],
        defaultValue: false,
        reverse: false,
        canBooleanLogic: false,
    },
];
