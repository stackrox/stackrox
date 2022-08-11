import startCase from 'lodash/startCase';

import {
    portExposureLabels,
    envVarSrcLabels,
    rbacPermissionLabels,
    policyCriteriaCategories,
    mountPropagationLabels,
    seccompProfileTypeLabels,
    severityRatings,
} from 'messages/common';
import { FeatureFlagEnvVar } from 'types/featureFlag';

const equalityOptions: DescriptorOption[] = [
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

const cpuResource = (label: string): GroupDescriptor => ({
    label,
    name: startCase(label),
    shortName: label,
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

const capabilities: DescriptorOption[] = [
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

const APIVerbs: DescriptorOption[] = ['CREATE', 'DELETE', 'GET', 'PATCH', 'UPDATE'].map((verb) => ({
    label: verb,
    value: verb,
}));

const memoryResource = (label: string): GroupDescriptor => ({
    label,
    name: startCase(label),
    shortName: label,
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

// TODO Delete after signaturePolicyCriteria type encapsulates its behavior.
export const imageSigningCriteriaName = 'Image Signature Verified By';

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
        featureFlagDependency: 'ROX_WHATEVER',
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
    featureFlagDependency: optional property to filter descriptor by feature flag enabled or disabled
 */

export type DescriptorOption = {
    label: string;
    value: string;
};

export type SubComponent = {
    type: 'number' | 'select' | 'text'; // add more if needed
    options?: DescriptorOption[];
    subpath: string;
    placeholder?: string;
    label?: string;
    min?: number;
    max?: number;
    step?: number;
};

export type BaseDescriptor = {
    label?: string;
    name: string;
    longName?: string;
    shortName?: string;
    negatedName?: string;
    category: string;
    type: DescriptorType;
    canBooleanLogic?: boolean;
    disabled?: boolean;
    featureFlagDependency?: FeatureFlagEnvVar;
};

export type DescriptorType =
    | 'group'
    | 'multiselect'
    | 'number'
    | 'radioGroup'
    | 'radioGroupString'
    | 'select'
    | 'text'
    | 'signaturePolicyCriteria';

export type Descriptor =
    | GroupDescriptor
    | NumberDescriptor
    | RadioGroupDescriptor
    | SelectDescriptor
    | TextDescriptor
    | SignatureDescriptor;

export type SignatureDescriptor = {
    type: 'signaturePolicyCriteria';
} & BaseDescriptor;

export type GroupDescriptor = {
    type: 'group';
    subComponents: SubComponent[];
    default?: boolean;
} & BaseDescriptor;

export type NumberDescriptor = {
    type: 'number';
    placeholder?: string;
} & BaseDescriptor;

export type RadioGroupDescriptor = {
    type: 'radioGroup' | 'radioGroupString';
    radioButtons: { text: string; value: string | boolean }[];
    defaultValue?: string | boolean;
    reverse?: boolean;
} & BaseDescriptor;

export type SelectDescriptor = {
    type: 'multiselect' | 'select';
    options: DescriptorOption[];
    placeholder?: string;
    reverse?: boolean;
} & BaseDescriptor;

export type TextDescriptor = {
    type: 'text';
    placeholder?: string;
} & BaseDescriptor;

export const policyConfigurationDescriptor: Descriptor[] = [
    {
        label: 'Image registry',
        name: 'Image Registry',
        shortName: 'Image registry',
        longName: 'Image pulled from registry',
        negatedName: 'Image not pulled from registry',
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'docker.io',
        canBooleanLogic: true,
    },
    {
        label: 'Image remote',
        name: 'Image Remote',
        shortName: 'Image remote',
        longName: 'Image name in the registry',
        negatedName: `Image name in the registry doesn't match`,
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'library/nginx',
        canBooleanLogic: true,
    },
    {
        label: 'Image tag',
        name: 'Image Tag',
        shortName: 'Image tag',
        negatedName: `Image tag doesn't match`,
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'latest',
        canBooleanLogic: true,
    },
    {
        label: 'Not verified by trusted image signers',
        name: imageSigningCriteriaName,
        shortName: 'Not verified by trusted image signers',
        longName: 'Not verified by trusted image signers',
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'signaturePolicyCriteria',
        canBooleanLogic: true,
    },
    {
        label: 'Days since image was created',
        name: 'Image Age',
        shortName: 'Image age',
        longName: 'Minimum days since image was built',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'number',
        placeholder: '1',
        canBooleanLogic: false,
    },
    {
        label: 'Days since image was last scanned',
        name: 'Image Scan Age',
        shortName: 'Image scan age',
        longName: 'Minimum days since last image scan',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'number',
        placeholder: '1',
        canBooleanLogic: false,
    },
    {
        label: 'Image user',
        name: 'Image User',
        shortName: 'Image user',
        negatedName: `Image user is not`,
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'text',
        placeholder: '0',
        canBooleanLogic: false,
    },
    {
        label: 'Dockerfile line',
        name: 'Dockerfile Line',
        shortName: 'Dockerfile line',
        longName: 'Disallowed dockerfile line',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'group',
        subComponents: [
            {
                type: 'select',
                options: [
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
        label: 'Image is NOT scanned',
        name: 'Unscanned Image',
        shortName: 'Unscanned image',
        longName: 'Image scan status',
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
        longName: 'Common Vulnerability Scoring System (CVSS) score',
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
                max: 10.0,
                min: 0.0,
                step: 0.1,
                subpath: 'value',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Severity',
        name: 'Severity',
        longName: 'Vulnerability severity rating',
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
                placeholder: 'Select a severity',
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
        label: 'Fixed by',
        name: 'Fixed By',
        shortName: 'Fixed by',
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
        label: 'Image component',
        name: 'Image Component',
        shortName: 'Image component',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'group',
        subComponents: [
            {
                type: 'text',
                label: 'Component name',
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
        longName: 'Image operating system',
        negatedName: `Image operating system doesn't match`,
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'text',
        placeholder: 'ubuntu:19.04',
        canBooleanLogic: true,
    },
    {
        label: 'Environment variable',
        name: 'Environment Variable',
        shortName: 'Environment variable',
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
                label: 'Value from',
                placeholder: 'Select one',
                subpath: 'source',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Disallowed annotation',
        name: 'Disallowed Annotation',
        shortName: 'Disallowed annotation',
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
        label: 'Required label',
        name: 'Required Label',
        shortName: 'Required label',
        longName: 'Required deployment label',
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
        label: 'Required annotation',
        name: 'Required Annotation',
        shortName: 'Required annotation',
        longName: 'Required deployment annotation',
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
        label: 'Runtime class',
        name: 'Runtime Class',
        shortName: 'Runtime class',
        negatedName: `Runtime class doesn't match`,
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'text',
        placeholder: 'kata',
        canBooleanLogic: true,
    },
    {
        label: 'Volume name',
        name: 'Volume Name',
        shortName: 'Volume name',
        negatedName: `Volume name doesn't match`,
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: 'docker-socket',
        canBooleanLogic: true,
    },
    {
        label: 'Volume source',
        name: 'Volume Source',
        shortName: 'Volume source',
        negatedName: `Volume source doesn't match`,
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: '/var/run/docker.sock',
        canBooleanLogic: true,
    },
    {
        label: 'Volume destination',
        name: 'Volume Destination',
        shortName: 'Volume destination',
        negatedName: `Volume destination doesn't match`,
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: '/var/run/docker.sock',
        canBooleanLogic: true,
    },
    {
        label: 'Volume type',
        name: 'Volume Type',
        shortName: 'Volume type',
        negatedName: `Volume type doesn't match`,
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: 'bind, secret',
        canBooleanLogic: true,
    },
    {
        label: 'Writable mounted volume',
        name: 'Writable Mounted Volume',
        shortName: 'Writable mounted volume',
        longName: 'Mounted volume writability',
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
        name: 'Mount Propagation',
        shortName: 'Mount propagation',
        negatedName: 'Mount propagation is not',
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
        shortName: 'Exposed port protocol',
        negatedName: `Exposed port protocol doesn't match`,
        category: policyCriteriaCategories.NETWORKING,
        type: 'text',
        placeholder: 'tcp',
        canBooleanLogic: true,
    },
    {
        label: 'Exposed node port',
        name: 'Exposed Node Port',
        shortName: 'Exposed node port',
        negatedName: `Exposed node port doesn't match`,
        category: policyCriteriaCategories.NETWORKING,
        type: 'text',
        placeholder: '22',
        canBooleanLogic: true,
    },
    {
        label: 'Port',
        name: 'Exposed Port',
        shortName: 'Exposed port',
        negatedName: `Exposed port doesn't match`,
        category: policyCriteriaCategories.NETWORKING,
        type: 'number',
        placeholder: '22',
        canBooleanLogic: true,
    },
    {
        label: 'Port Exposure',
        name: 'Port Exposure Method',
        shortName: 'Port exposure method',
        negatedName: 'Port exposure method is not',
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
        label: 'Network baselining enabled',
        name: 'Unexpected Network Flow Detected',
        shortName: 'Unexpected network flow detected',
        longName: 'Network baselining status',
        category: policyCriteriaCategories.NETWORKING,
        type: 'radioGroup',
        radioButtons: [
            { text: 'Unexpected network flow', value: true },
            { text: 'Expected network flow', value: false },
        ],
        defaultValue: false,
        reverse: false,
        canBooleanLogic: false,
    },
    {
        label: 'Ingress Network Policy',
        name: 'Has Ingress Network Policy',
        shortName: 'Ingress Network Policy',
        longName: 'Ingress Network Policy',
        category: policyCriteriaCategories.NETWORKING,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Policy Missing',
                value: false,
            },
            {
                text: 'Policy Present',
                value: true,
            },
        ],
        defaultValue: false,
        canBooleanLogic: false,
        featureFlagDependency: 'ROX_NETPOL_FIELDS',
    },
    {
        label: 'Egress Network Policy',
        name: 'Has Egress Network Policy',
        shortName: 'Egress Network Policy',
        longName: 'Egress Network Policy',
        category: policyCriteriaCategories.NETWORKING,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Policy Missing',
                value: false,
            },
            {
                text: 'Policy Present',
                value: true,
            },
        ],
        defaultValue: false,
        canBooleanLogic: false,
        featureFlagDependency: 'ROX_NETPOL_FIELDS',
    },
    cpuResource('Container CPU request'),
    cpuResource('Container CPU limit'),
    memoryResource('Container memory request'),
    memoryResource('Container memory limit'),
    {
        label: 'Privileged',
        name: 'Privileged Container',
        shortName: 'Privileged container',
        longName: 'Privileged container status',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Privileged container',
                value: true,
            },
            {
                text: 'Not a privileged container',
                value: false,
            },
        ],
        defaultValue: true,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        label: 'Read-only root filesystem',
        name: 'Read-Only Root Filesystem',
        shortName: 'Read-only root filesystem',
        longName: 'Root filesystem writability',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Read-only',
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
        label: 'Seccomp profile type',
        name: 'Seccomp Profile Type',
        shortName: 'Seccomp profile type',
        negatedName: 'Seccomp profile type is not',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'radioGroupString',
        radioButtons: Object.keys(seccompProfileTypeLabels).map((key) => ({
            text: seccompProfileTypeLabels[key],
            value: key,
        })),
        canBooleanLogic: false,
    },
    {
        label: 'Privilege escalation',
        name: 'Allow Privilege Escalation',
        shortName: 'Privilege escalation',
        longName: 'Privilege escalation on container',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Allowed',
                value: true,
            },
            {
                text: 'Not allowed',
                value: false,
            },
        ],
        defaultValue: true,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        label: 'Share host network namespace',
        name: 'Host Network',
        shortName: 'Host network',
        longName: 'Host network',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Uses host network namespace',
                value: true,
            },
            {
                text: 'Does not use host network namespace',
                value: false,
            },
        ],
        defaultValue: true,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        label: 'Share host PID namespace',
        name: 'Host PID',
        longName: 'Host PID',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Uses host PID namespace',
                value: true,
            },
            {
                text: 'Does not use host PID namespace',
                value: false,
            },
        ],
        defaultValue: true,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        label: 'Share host IPC namespace',
        name: 'Host IPC',
        longName: 'Host IPC',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Uses host IPC namespace',
                value: true,
            },
            {
                text: 'Does not use host IPC namespace',
                value: false,
            },
        ],
        defaultValue: true,
        disabled: true,
        canBooleanLogic: false,
    },
    {
        name: 'Drop Capabilities',
        shortName: 'Drop capabilities',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'select',
        options: [...capabilities],
        canBooleanLogic: true,
    },
    {
        name: 'SBOM Verification Status',
        shortName: 'Image SBOM',
        longName: 'Image SBOM coverage',
        label: 'The SBOM must cover the image at least',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'select',
        options: [
            {
                label: 'SBOM covers the whole image',
                value: 'COVERED',
            },
        ],
    },
    {
        name: 'Add Capabilities',
        shortName: 'Add capabilities',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'select',
        options: [...capabilities],
        canBooleanLogic: true,
    },
    {
        name: 'Process Name',
        shortName: 'Process name',
        negatedName: `Process name doesn't match`,
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'text',
        placeholder: 'apt-get',
        canBooleanLogic: true,
    },
    {
        name: 'Process Ancestor',
        shortName: 'Process ancestor',
        negatedName: `Process ancestor doesn't match`,
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'text',
        placeholder: 'java',
        canBooleanLogic: true,
    },
    {
        name: 'Process Arguments',
        shortName: 'Process arguments',
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
        name: 'Writable Host Mount',
        shortName: 'Writable host mount',
        longName: 'Host mount writability',
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
        label: 'Process baselining enabled',
        name: 'Unexpected Process Executed',
        shortName: 'Unexpected process executed',
        longName: 'Process baselining status',
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'radioGroup',
        radioButtons: [
            { text: 'Unexpected process', value: true },
            { text: 'Expected process', value: false },
        ],
        defaultValue: false,
        reverse: false,
        canBooleanLogic: false,
    },
    {
        label: 'Service account',
        name: 'Service Account',
        shortName: 'Service account',
        longName: 'Service account name',
        negatedName: `Service account name doesn't match`,
        category: policyCriteriaCategories.KUBERNETES_ACCESS,
        type: 'text',
        canBooleanLogic: true,
    },
    {
        label: 'Automount service account token',
        name: 'Automount Service Account Token',
        shortName: 'Automount service account token',
        longName: 'Automount service account token',
        category: policyCriteriaCategories.KUBERNETES_ACCESS,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Automount',
                value: true,
            },
            {
                text: 'Do not mount',
                value: false,
            },
        ],
        defaultValue: false,
        canBooleanLogic: false,
    },
    {
        label: 'Minimum RBAC permissions',
        name: 'Minimum RBAC Permissions',
        shortName: 'Minimum RBAC permissions',
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
        label: 'Required image label',
        name: 'Required Image Label',
        shortName: 'Require image label',
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
        label: 'Disallowed image label',
        name: 'Disallowed Image Label',
        shortName: 'Disallowed image label',
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
        label: 'Container name',
        name: 'Container Name',
        shortName: 'Container name',
        longName: 'Container name',
        negatedName: `Container name doesn't match`,
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'text',
        placeholder: 'default',
        canBooleanLogic: true,
    },
    {
        label: 'AppArmor profile',
        name: 'AppArmor Profile',
        shortName: 'AppArmor profile',
        longName: 'AppArmor profile',
        negatedName: `AppArmor profile doesn't match`,
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'text',
        placeholder: 'default',
        canBooleanLogic: true,
    },
    {
        label: 'Kubernetes action',
        name: 'Kubernetes Resource',
        longName: 'Kubernetes action',
        shortName: 'Kubernetes action',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'select',
        options: [
            {
                label: 'Pod exec',
                value: 'PODS_EXEC',
            },
            {
                label: 'Pods port forward',
                value: 'PODS_PORTFORWARD',
            },
        ],
        canBooleanLogic: false,
    },
    {
        label: 'Kubernetes user name',
        name: 'Kubernetes User Name',
        shortName: 'Kubernetes user name',
        negatedName: "Kubernetes user name doesn't match",
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'Kubernetes user groups',
        name: 'Kubernetes User Groups',
        shortName: 'Kubernetes user groups',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'Replicas',
        name: 'Replicas',
        shortName: 'Replicas',
        longName: 'Replicas',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'group',
        subComponents: [
            {
                type: 'select',
                options: equalityOptions,
                subpath: 'key',
            },
            {
                type: 'number',
                placeholder: '# of replicas',
                min: 0,
                step: 1,
                subpath: 'value',
            },
        ],
        canBooleanLogic: true,
    },
    {
        label: 'Liveness probe',
        name: 'Liveness Probe Defined',
        shortName: 'Liveness probe',
        longName: 'Liveness probe',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Defined',
                value: true,
            },
            {
                text: 'Not defined',
                value: false,
            },
        ],
        defaultValue: false,
        canBooleanLogic: false,
    },
    {
        label: 'Readiness probe',
        name: 'Readiness Probe Defined',
        shortName: 'Readiness probe',
        longName: 'Readiness probe',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'Defined',
                value: true,
            },
            {
                text: 'Not defined',
                value: false,
            },
        ],
        defaultValue: false,
        canBooleanLogic: false,
    },
];

export const auditLogDescriptor: Descriptor[] = [
    {
        label: 'Kubernetes resource',
        name: 'Kubernetes Resource',
        shortName: 'Kubernetes resource',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'select',
        placeholder: 'Select a resource',
        options: [
            {
                label: 'Config maps',
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
        label: 'Kubernetes API verb',
        name: 'Kubernetes API Verb',
        shortName: 'Kubernetes API verb',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'select',
        placeholder: 'Select an API verb',
        options: APIVerbs,
        canBooleanLogic: false,
    },
    {
        label: 'Kubernetes resource name',
        name: 'Kubernetes Resource Name',
        shortName: 'Kubernetes resource name',
        negatedName: "Kubernetes resource name doesn't match",
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'Kubernetes user name',
        name: 'Kubernetes User Name',
        shortName: 'Kubernetes user name',
        negatedName: "Kubernetes user name doesn't match",
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'Kubernetes user group',
        name: 'Kubernetes User Groups',
        shortName: 'Kubernetes user groups',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'User agent',
        name: 'User Agent',
        shortName: 'User agent',
        negatedName: "User agent doesn't match",
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'Source IP address',
        name: 'Source IP Address',
        shortName: 'Source IP address',
        negatedName: "Source IP address doesn't match",
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'text',
        canBooleanLogic: false,
    },
    {
        label: 'Is impersonated user',
        name: 'Is Impersonated User',
        shortName: 'Is impersonated user',
        category: policyCriteriaCategories.KUBERNETES_EVENTS,
        type: 'radioGroup',
        radioButtons: [
            { text: 'True', value: true },
            { text: 'False', value: false },
        ],
        canBooleanLogic: false,
    },
];
