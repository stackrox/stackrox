import {
    envVarSrcLabels,
    mountPropagationLabels,
    policyCriteriaCategories,
    portExposureLabels,
    rbacPermissionLabels,
    seccompProfileTypeLabels,
    severityRatings,
} from 'messages/common';
import type { FeatureFlagEnvVar } from 'types/featureFlag';
import type { LifecycleStage } from 'types/policy.proto';

import ImageSigningTableModal from './ImageSigningTableModal';

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

const subComponentsForContainerCPU: SubComponent[] = [
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
];

const dropCapabilities: DescriptorOption[] = [
    'ALL',
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

const addCapabilities: DescriptorOption[] = [
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

const fileOperationOptions: DescriptorOption[] = [
    'OPEN',
    'CREATE',
    'UNLINK',
    'RENAME',
    'PERMISSION_CHANGE',
    'OWNERSHIP_CHANGE',
].map((operation) => ({ label: operation, value: operation }));

const fileActivityPathOptions: DescriptorOption[] = [
    '/etc/passwd',
    '/etc/ssh/sshd_config',
    '/etc/shadow',
    '/etc/sudoers',
].map((path) => ({ label: path, value: path }));

const subComponentsForContainerMemory: SubComponent[] = [
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
];

// TODO Delete after signaturePolicyCriteria type encapsulates its behavior.
export const imageSigningCriteriaName = 'Image Signature Verified By';
export const mountPropagationCriteriaName = 'Mount Propagation';

// A form descriptor for every option (key) on the policy criteria form page.
/*
    e.g.
    {
        label: 'Image Tag',
        name: 'Image Tag',
        negatedName: 'Image tag doesn’t match',
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'latest',
        canBooleanLogic: true,
        featureFlagDependency: 'ROX_WHATEVER',
    },

    label: for legacy policy alert labels
    name: corresponds to backend field names for id or key props but not for visible text
          https://github.com/stackrox/stackrox/blob/master/pkg/booleanpolicy/fieldnames/list.go

    Add allowed items if unit tests fail for new rules that follow the rules (pardon pun).
    Sentence case for these names, which cannot be equal to each other:
    shortName: required string for title of rule criterion in group of form and tree of modal
    longName: optional string for subtitle of rule criterion in group of form and tree of modal
    negatedName: optional string used to display UI when negated
        (if this does not exist, the UI assumes that the field cannot be negated)

    category: the category grouping for the policy criteria (collapsible group in keys)
    type: the type of form field to render when dragged to the Policy Field Card
    subComponents: subfields the field renders when dragged to Policy Field Card if 'group' type
    radioButtons: button options if 'radioGroup' or 'radioGroupString' type
    options: options if 'select' or 'multiselect' or 'multiselect-creatable' type
    placeholder: string to display as placeholder if applicable
    canBooleanLogic: indicates whether the field supports the AND/OR boolean operator
        (UI assumes false by default)
    defaultValue: the default value to set, if provided
    disabled: disables the field entirely
    reverse: will reverse boolean value on store
    infoText: optional short text for title of info Alert element in card body of policy field in wizard
    featureFlagDependency: optional property to filter descriptor by feature flags enabled or disabled
 */

export type DescriptorOption = {
    label: string;
    value: string;
};

export type SubComponent = TextSubComponent | NumberSubComponent | SelectSubComponent;

export type BaseSubComponent = {
    subpath: string;
    label?: string;
    placeholder?: string;
};

export type TextSubComponent = {
    type: 'text';
} & BaseSubComponent;

export type NumberSubComponent = {
    type: 'number';
    min: number;
    max?: number;
    step?: number;
} & BaseSubComponent;

export type SelectSubComponent = {
    type: 'select';
    options: DescriptorOption[];
} & BaseSubComponent;

type BaseDescriptor = {
    label?: string;
    name: string;
    shortName: string;
    longName?: string;
    infoText?: string;
    description?: string;
    category: string;
    type: DescriptorType;
    disabled?: boolean;
    featureFlagDependency?: FeatureFlagEnvVar[];
    lifecycleStages: LifecycleStage[];
};

type DescriptorCanBoolean = {
    canBooleanLogic: boolean;
};

type DescriptorCanNotBoolean = {
    canBooleanLogic: false;
};

type DescriptorCanNegate = {
    negatedName?: string;
};

export type DescriptorType =
    | 'group'
    | 'multiselect'
    | 'number'
    | 'radioGroup'
    | 'radioGroupString'
    | 'select'
    | 'text'
    | 'tableModal';

export type Descriptor =
    | GroupDescriptor
    | NumberDescriptor
    | RadioGroupDescriptor
    | RadioGroupStringDescriptor
    | SelectDescriptor
    | TextDescriptor
    | TableModalDescriptor;

export type TableModalDescriptor = {
    type: 'tableModal';
    component: typeof ImageSigningTableModal;
    tableType: string;
} & BaseDescriptor &
    DescriptorCanBoolean;

export type GroupDescriptor = {
    type: 'group';
    subComponents: SubComponent[];
    default?: boolean;
} & BaseDescriptor &
    DescriptorCanBoolean;

export type NumberDescriptor = {
    type: 'number';
    placeholder?: string;
} & BaseDescriptor &
    DescriptorCanBoolean &
    DescriptorCanNegate;

type RadioButtonFalse = { text: string; value: false };
type RadioButtonTrue = { text: string; value: true };

export type RadioGroupDescriptor = {
    type: 'radioGroup';
    // radioButtons: { text: string; value: boolean }[];
    radioButtons: [RadioButtonFalse, RadioButtonTrue] | [RadioButtonTrue, RadioButtonFalse];
    defaultValue?: boolean; // TODO missing only in 'Is Impersonated User'
    reverse?: boolean; // TODO what are pro and con to require it?
} & BaseDescriptor &
    DescriptorCanNotBoolean;

export type RadioGroupStringDescriptor = {
    type: 'radioGroupString';
    radioButtons: { text: string; value: string }[];
    // defaultValue?: string;
} & BaseDescriptor &
    DescriptorCanNotBoolean &
    DescriptorCanNegate;

export type SelectDescriptor = {
    type: 'multiselect' | 'select';
    options: DescriptorOption[];
    placeholder?: string;
    reverse?: boolean;
} & BaseDescriptor &
    DescriptorCanBoolean &
    DescriptorCanNegate;

export type TextDescriptor = {
    type: 'text';
    placeholder?: string;
} & BaseDescriptor &
    DescriptorCanBoolean &
    DescriptorCanNegate;

export const policyCriteriaDescriptors: Descriptor[] = [
    {
        label: 'Image registry',
        name: 'Image Registry',
        shortName: 'Image registry',
        longName: 'Container registry name is',
        negatedName: 'Container registry name is not',
        description: 'Triggers a violation when the image registry name matches (or doesn\'t match if negated) the specified pattern. The registry is the domain where the container image is hosted (e.g., quay.io, docker.io, gcr.io). Regular expressions are supported. Example: specifying ".*docker.io.*" triggers violations for all images from Docker Hub. Use this to restrict images to approved registries or block untrusted sources.',
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'for example: quay.io',
        canBooleanLogic: true,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Image name',
        name: 'Image Remote',
        shortName: 'Image name',
        longName: 'Image name is',
        negatedName: 'Image name doesn\'t match',
        description: 'Triggers a violation when the image name (repository path) matches (or doesn\'t match if negated) the specified pattern. This is the namespace/repository portion without the registry domain or tag (e.g., library/nginx, stackrox/main). Regular expressions are supported. Example: ".*nginx.*" triggers violations for all nginx-related images. Use this to restrict which specific images or image families can be deployed.',
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'library/nginx',
        canBooleanLogic: true,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Image tag',
        name: 'Image Tag',
        shortName: 'Image tag',
        longName: 'Image tag is',
        negatedName: 'Image tag doesn\'t match',
        description: 'Triggers a violation when the image tag matches (or doesn\'t match if negated) the specified pattern. Tags identify specific versions or variants of an image (e.g., latest, v1.2.3, stable). Regular expressions are supported. Example: specifying "^latest$" triggers violations for images using the "latest" tag. Common uses include blocking mutable tags like "latest" to ensure reproducible deployments, or requiring semantic versioning patterns.',
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'latest',
        canBooleanLogic: true,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Require image signature',
        name: imageSigningCriteriaName,
        shortName: 'Require image signature',
        longName: 'Image must be signed by a trusted signer',
        description: 'Triggers a violation when an image lacks a valid cryptographic signature verifiable by at least one of the specified signature integrations. This criterion fires if: (1) the image has no signature at all, OR (2) the image\'s signature cannot be verified by any of the configured signers. This ensures images come from trusted sources and haven\'t been tampered with. Supports OR logic to allow multiple trusted signers. Example: if you specify "Cosign-Prod" OR "Cosign-Dev", the violation occurs only if neither integration can verify the signature.',
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'tableModal',
        tableType: 'imageSigning',
        component: ImageSigningTableModal,
        canBooleanLogic: true,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Days since image was created',
        name: 'Image Age',
        shortName: 'Image age',
        longName: 'Minimum days since image was built',
        description: 'Triggers a violation when an image was created/built MORE than the specified number of days ago. For example, setting this to 30 means violations occur for images older than 30 days. This helps identify stale images that may contain outdated software or unpatched vulnerabilities. Use this to enforce that only recently built images are deployed.',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'number',
        placeholder: '1',
        canBooleanLogic: false,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Days since image was last scanned',
        name: 'Image Scan Age',
        shortName: 'Image scan age',
        longName: 'Minimum days since last image scan',
        description: 'Triggers a violation when an image was last scanned for vulnerabilities MORE than the specified number of days ago. For example, setting this to 7 means violations occur if the image hasn\'t been scanned in over a week. This ensures images are regularly re-scanned as new vulnerabilities are discovered. Use this to maintain up-to-date vulnerability assessments.',
        category: policyCriteriaCategories.IMAGE_SCANNING,
        type: 'number',
        placeholder: '1',
        canBooleanLogic: false,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Image user',
        name: 'Image User',
        shortName: 'Image user',
        longName: 'USER directive in the Dockerfile is',
        negatedName: 'USER directive in the Dockerfile is not',
        description: 'Triggers a violation when the USER directive in the Dockerfile matches (or doesn\'t match if negated) the specified pattern. This can be a username, UID, or UID:GID format. Regular expressions are supported. Example: specifying "^0$" triggers violations for containers running as root (UID 0). Common use: enforce the principle of least privilege by blocking root users.',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'text',
        placeholder: '0',
        canBooleanLogic: false,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Dockerfile line',
        name: 'Dockerfile Line',
        shortName: 'Dockerfile line',
        longName: 'Disallowed dockerfile line',
        description: 'Matches specific lines in the Dockerfile based on both the instruction type (e.g., RUN, CMD, EXPOSE) and its arguments. Regular expressions are supported for the arguments field. This allows enforcement of Dockerfile best practices, such as preventing specific commands, disallowing certain exposed ports, or ensuring proper build instructions. For example, you can block "RUN apt-get" commands or flag dangerous EXPOSE directives.',
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
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Image scan status',
        name: 'Unscanned Image',
        shortName: 'Image scan status',
        longName: 'Image scan status is',
        description: 'Checks whether an image has been scanned for vulnerabilities by the integrated scanner. Selecting "Not scanned" triggers the policy for images that have never been scanned, which may occur if the image is from an unsupported registry or if scanning failed. Use this to ensure all deployed images have undergone vulnerability assessment.',
        category: policyCriteriaCategories.IMAGE_SCANNING,
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
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'CVSS',
        name: 'CVSS',
        shortName: 'CVSS',
        longName: 'Common Vulnerability Scoring System (CVSS) score',
        description: 'Matches images containing vulnerabilities with CVSS scores meeting specified criteria (greater than, less than, or equal to a value between 0-10). CVSS is an industry-standard vulnerability scoring system where higher scores indicate more severe vulnerabilities. This field uses the highest CVSS score from any vendor source (NVD, Red Hat, etc.). Combine with AND/OR logic to create flexible vulnerability severity policies.',
        category: policyCriteriaCategories.IMAGE_SCANNING,
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
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'NVD CVSS',
        name: 'NVD CVSS',
        shortName: 'NVD CVSS',
        longName:
            'Common Vulnerability Scoring System (CVSS) score from National Vulnerability Database (NVD)',
        infoText: 'NVD CVSS scores require Scanner V4',
        description: 'Matches images containing vulnerabilities with CVSS scores from the National Vulnerability Database (NVD) specifically, supporting comparison operators (greater than, less than, or equal to). Unlike the general CVSS field which uses the highest score from any vendor, this field uses only NVD scores. Requires Scanner V4. Useful when you want to enforce policies based specifically on NIST-maintained vulnerability assessments.',
        category: policyCriteriaCategories.IMAGE_SCANNING,
        type: 'group',
        subComponents: [
            {
                type: 'select',
                options: equalityOptions, // see nonStandardNumberFields
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
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Severity',
        name: 'Severity',
        shortName: 'Severity',
        longName: 'Vulnerability severity rating is',
        description: 'Matches images with vulnerabilities at specific severity levels (Low, Moderate, Important, or Critical) derived from CVSS scores or vendor-provided ratings. Supports comparison operators to match severities greater than, less than, or equal to the specified level. For example, ">= Important" matches both Important and Critical vulnerabilities. Severity levels provide a simpler alternative to numeric CVSS scores for policy creation.',
        category: policyCriteriaCategories.IMAGE_SCANNING,
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
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Fixable',
        name: 'Fixable',
        shortName: 'Fixable',
        description: 'Filters vulnerabilities based on whether a fix is available from the package maintainer. "CVE is fixable" triggers when a patched version exists that resolves the vulnerability, enabling you to prioritize remediating vulnerabilities that can actually be fixed. "CVE is not yet fixable" identifies vulnerabilities awaiting upstream patches, useful for tracking issues that require workarounds or acceptance.',
        category: policyCriteriaCategories.IMAGE_SCANNING,
        type: 'radioGroup',
        radioButtons: [
            {
                text: 'CVE is fixable',
                value: true,
            },
            {
                text: 'CVE is not yet fixable',
                value: false,
            },
        ],
        defaultValue: true,
        canBooleanLogic: false,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Fixed by',
        name: 'Fixed By',
        shortName: 'Fixed by',
        longName: 'Package version where a vulnerability is fixed',
        negatedName: 'Package version where a vulnerability is not fixed',
        description: 'Matches the specific package version string that fixes a vulnerability. Regular expressions are supported, allowing pattern matching across version formats. This criterion is typically used with other vulnerability criteria (like CVE) to identify whether a fix is available in a specific version range. For example, you can check if a fix exists in versions matching a pattern like "2.3.*" or later.',
        category: policyCriteriaCategories.IMAGE_SCANNING,
        type: 'text',
        placeholder: '.*',
        canBooleanLogic: true,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'CVE',
        name: 'CVE',
        shortName: 'CVE',
        longName: 'CVE identifier is',
        negatedName: 'CVE identifier doesn\'t match',
        description: 'Matches specific Common Vulnerabilities and Exposures (CVE) identifiers in the format CVE-YYYY-NNNNN (e.g., CVE-2017-11882). Regular expressions are supported, enabling you to match multiple CVEs with patterns. Use this to create policies targeting specific known vulnerabilities, track particular CVEs of concern, or exclude specific CVEs from policy violations. Combine with AND/OR logic for complex vulnerability policies.',
        category: policyCriteriaCategories.IMAGE_SCANNING,
        type: 'text',
        placeholder: 'CVE-2017-11882',
        canBooleanLogic: true,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Days Since CVE Was Published',
        name: 'Days Since CVE Was Published',
        shortName: 'Days since CVE was published',
        description: 'Triggers when a CVE has been publicly published for more than the specified number of days. This helps enforce service level agreements (SLAs) for patching known vulnerabilities by giving teams a grace period to apply fixes. For example, setting this to 30 days means the policy only triggers on CVEs that were disclosed more than a month ago, providing time for assessment and remediation.',
        category: policyCriteriaCategories.IMAGE_SCANNING,
        type: 'number',
        placeholder: '0',
        canBooleanLogic: false,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Days Since CVE Was First Discovered In Image',
        name: 'Days Since CVE Was First Discovered In Image',
        shortName: 'Days since CVE was first discovered in image',
        description: 'Triggers when StackRox first discovered a CVE in a specific image more than the specified number of days ago. This differs from publication date by tracking when you became aware of the vulnerability in your environment. Useful for creating image-specific remediation SLAs and ensuring vulnerable images are eventually rebuilt or replaced.',
        category: policyCriteriaCategories.IMAGE_SCANNING,
        type: 'number',
        placeholder: '0',
        canBooleanLogic: false,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Days Since CVE Was First Discovered In System',
        name: 'Days Since CVE Was First Discovered In System',
        shortName: 'Days since CVE was first discovered in system',
        description: 'Triggers when StackRox first discovered a CVE anywhere across all clusters and deployments more than the specified number of days ago. This provides a system-wide perspective on vulnerability age, regardless of which specific image or deployment it was first seen in. Use this to enforce organization-wide remediation timelines for known vulnerabilities.',
        category: policyCriteriaCategories.IMAGE_SCANNING,
        type: 'number',
        placeholder: '0',
        canBooleanLogic: false,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Image component',
        name: 'Image Component',
        shortName: 'Image component',
        description: 'Matches software components (packages, libraries) present in an image by name and optionally by version. Regular expressions are supported for both fields. Use this to enforce policies about allowed or prohibited software packages, such as banning specific libraries, requiring certain versions, or flagging deprecated components. The version field can be left empty to match any version of a component.',
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
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Image OS',
        name: 'Image OS',
        shortName: 'Image OS',
        longName: 'Image operating system name and version is',
        negatedName: 'Image operating system name and version doesn\'t match',
        description: 'Matches the base operating system of the image including its version, typically in the format "os_name:version" (e.g., alpine:3.17.3, ubuntu:20.04, rhel:8.5). Regular expressions are supported. Use this to enforce standardization on approved OS distributions and versions, flag end-of-life operating systems, or ensure compliance with organizational OS policies.',
        category: policyCriteriaCategories.IMAGE_CONTENTS,
        type: 'text',
        placeholder: 'for example: alpine:3.17.3',
        canBooleanLogic: true,
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Environment variable',
        name: 'Environment Variable',
        shortName: 'Environment variable',
        description: 'Matches environment variables by key (name), value, and source type. Sources include raw values (directly specified in YAML), ConfigMap references, Secret references, field references, and resource field references. Regular expressions are supported for key and value when using raw source. Use this to enforce policies about sensitive data exposure, required configurations, or prohibited environment settings. Note that for non-raw sources, only the key is evaluated.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Disallowed annotation',
        name: 'Disallowed Annotation',
        shortName: 'Disallowed annotation',
        description: 'Triggers when a Kubernetes deployment has annotations matching the specified key and optionally value. Regular expressions are supported for both fields. The value can be left empty to match any value for the key. Use this to prevent specific annotations that may indicate misconfigurations, deprecated practices, or security risks. Annotations are Kubernetes metadata that don\'t affect runtime behavior but provide information to tools and operators.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Required label',
        name: 'Required Label',
        shortName: 'Required label',
        longName: 'Required deployment label',
        description: 'Triggers when a Kubernetes deployment is missing labels matching the specified key and optionally value pattern. Regular expressions are supported for both fields. Use this to enforce labeling standards for resource organization, cost allocation, ownership tracking, or environment identification. The value can be a pattern like ".*" to require the label exists with any value, or a specific pattern to validate label values.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Required annotation',
        name: 'Required Annotation',
        shortName: 'Required annotation',
        longName: 'Required deployment annotation',
        description: 'Triggers when a Kubernetes deployment is missing annotations matching the specified key and optionally value pattern. Regular expressions are supported for both fields. Annotations provide metadata for tools and operators. Use this to enforce documentation standards, ensure integration configurations are present, or validate that required metadata like contact information or change tracking IDs are included.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Runtime class',
        name: 'Runtime Class',
        shortName: 'Runtime class',
        longName: 'Privilege escalation on container is',
        negatedName: 'Privilege escalation on container doesn\'t match',
        description: 'Triggers a violation when the RuntimeClass of the deployment matches (or doesn\'t match if negated) the specified pattern. RuntimeClass determines which container runtime handles the workload (e.g., kata for Kata Containers, gvisor for gVisor). Regular expressions are supported. Example: specifying "kata" triggers violations for deployments using Kata Containers runtime. Use this to enforce or restrict specific runtime security boundaries.',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'text',
        placeholder: 'kata',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Volume name',
        name: 'Volume Name',
        shortName: 'Volume name',
        longName: 'Volume name is',
        negatedName: 'Volume name doesn\'t match',
        description: 'Triggers a violation when a mounted volume\'s name matches (or doesn\'t match if negated) the specified pattern. Regular expressions are supported. Example: specifying ".*socket.*" triggers violations for any volume with "socket" in its name. Use this to identify potentially dangerous volume mounts like docker-socket or to enforce volume naming conventions.',
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: 'docker-socket',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Volume source',
        name: 'Volume Source',
        shortName: 'Volume source path',
        longName: 'Volume source path is',
        negatedName: 'Volume source doesn\'t match',
        description: 'Triggers a violation when a volume\'s source path on the host matches (or doesn\'t match if negated) the specified pattern. For hostPath volumes, this is the path on the host filesystem. Regular expressions are supported. Example: specifying "^/var/run/docker\\.sock$" triggers violations for containers mounting the Docker socket. Use this to prevent access to sensitive host paths.',
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: '/var/run/docker.sock',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Volume destination',
        name: 'Volume Destination',
        shortName: 'Volume destination path',
        longName: 'Volume destination (mountPath) path is',
        negatedName: 'Volume destination doesn\'t match',
        description: 'Triggers a violation when a volume\'s mount path inside the container matches (or doesn\'t match if negated) the specified pattern. This is where the volume appears in the container\'s filesystem. Regular expressions are supported. Example: specifying "^/etc$" triggers violations for volumes mounted at /etc. Use this to prevent overwriting critical system directories or enforce mount path conventions.',
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: '/var/run/docker.sock',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Volume type',
        name: 'Volume Type',
        shortName: 'Volume type',
        longName: 'Volume type (e.g. secret, configMap, hostPath) is',
        negatedName: 'Volume type doesn\'t match',
        description: 'Triggers a violation when the volume type matches (or doesn\'t match if negated) the specified pattern. Volume types include hostPath, secret, configMap, persistentVolumeClaim, emptyDir, etc. Regular expressions are supported. Example: specifying "hostPath" triggers violations for any hostPath volumes. Use this to restrict dangerous volume types like hostPath that provide access to the host filesystem.',
        category: policyCriteriaCategories.STORAGE,
        type: 'text',
        placeholder: 'hostPath',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Mounted volume writability',
        name: 'Writable Mounted Volume',
        shortName: 'Mounted volume writability',
        longName: 'Mounted volume is',
        description: 'Triggers a violation based on whether volumes are mounted as writable or read-only. Selecting "Writable" triggers violations when ANY volume is mounted with write permissions. Selecting "Read-only" triggers when volumes are mounted read-only. Use "Writable" to enforce read-only mounts for security, preventing containers from modifying shared data or host paths.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        name: mountPropagationCriteriaName,
        shortName: 'Mount propagation',
        longName: 'Mount propagation is',
        negatedName: 'Mount propagation is not',
        description: 'Triggers a violation when volume mount propagation matches (or doesn\'t match if negated) the specified mode. Propagation modes are: None (mount changes don\'t propagate), HostToContainer (host mount changes propagate to container), or Bidirectional (mount changes propagate both ways). Example: selecting "Bidirectional" triggers violations for bidirectional mounts. Use this to prevent dangerous bidirectional propagation that could affect the host system.',
        category: policyCriteriaCategories.STORAGE,
        type: 'select',
        options: Object.keys(mountPropagationLabels).map((key) => ({
            label: mountPropagationLabels[key],
            value: key,
        })),
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Protocol',
        name: 'Exposed Port Protocol',
        shortName: 'Exposed port protocol',
        negatedName: 'Exposed port protocol doesn\'t match',
        description: 'Triggers a violation when the protocol used by an exposed port matches (or doesn\'t match if negated) the specified pattern (e.g., tcp, udp, sctp). Regular expressions are supported. Example: specifying "tcp" triggers violations for services exposing TCP ports. Use this in combination with port number criteria to identify specific network services.',
        category: policyCriteriaCategories.NETWORKING,
        type: 'text',
        placeholder: 'tcp',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Exposed node port',
        name: 'Exposed Node Port',
        shortName: 'Exposed node port',
        negatedName: 'Exposed node port doesn\'t match',
        description: 'Triggers a violation when a deployment exposes a NodePort matching (or not matching if negated) the specified value or pattern. NodePorts are externally accessible ports on cluster nodes (typically 30000-32767). Regular expressions and comparison operators are supported. Example: specifying "22" triggers violations for SSH exposed via NodePort. Use this to restrict external network exposure.',
        category: policyCriteriaCategories.NETWORKING,
        type: 'text',
        placeholder: '22',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Port',
        name: 'Exposed Port',
        shortName: 'Exposed port',
        negatedName: 'Exposed port doesn\'t match',
        description: 'Triggers a violation when a deployment exposes a port matching (or not matching if negated) the specified value or range. Supports comparison operators (>, >=, <, <=) and exact values. Example: ">1024" triggers violations for ports above 1024, or "22" for SSH port. Use this to identify services exposing well-known ports or to enforce port range policies.',
        category: policyCriteriaCategories.NETWORKING,
        type: 'text', // Use 'text' instead of 'number', as this field supports range qualifiers (>, >=, <, <=)
        placeholder: '22',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Port Exposure',
        name: 'Port Exposure Method',
        shortName: 'Port exposure method',
        negatedName: 'Port exposure method is not',
        description: 'Triggers a violation when ports are exposed using the specified method (or not using it if negated). Methods include: Route (OpenShift routes), LoadBalancer (cloud load balancers), NodePort (exposed on all nodes), HostPort (bound to host IP), or "Exposure type is not set". Example: selecting "LoadBalancer" triggers violations for services using cloud load balancers. Use this to control how services are exposed externally.',
        category: policyCriteriaCategories.NETWORKING,
        type: 'select',
        options: Object.keys(portExposureLabels)
            .filter((key) => key !== 'INTERNAL')
            .map((key) => ({
                label: portExposureLabels[key],
                value: key,
            })),
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Ingress Network Policy',
        name: 'Has Ingress Network Policy',
        shortName: 'Ingress network policy',
        description: 'Triggers a violation based on presence of Kubernetes ingress NetworkPolicy. Selecting "Policy Missing" triggers violations when NO ingress NetworkPolicy is defined for the deployment. Selecting "Policy Present" triggers when an ingress policy EXISTS. Use "Policy Missing" to enforce that all deployments have ingress traffic controls defined, implementing network segmentation and zero-trust networking.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Egress Network Policy',
        name: 'Has Egress Network Policy',
        shortName: 'Egress network policy',
        description: 'Triggers a violation based on presence of Kubernetes egress NetworkPolicy. Selecting "Policy Missing" triggers violations when NO egress NetworkPolicy is defined for the deployment. Selecting "Policy Present" triggers when an egress policy EXISTS. Use "Policy Missing" to enforce that all deployments have egress (outbound) traffic controls defined, preventing unauthorized data exfiltration.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Container CPU request',
        name: 'Container CPU Request',
        shortName: 'Container CPU request',
        description: 'Triggers a violation when the container CPU request meets the specified comparison criteria (e.g., >, >=, =, <=, <). CPU requests are measured in cores (fractional values allowed, e.g., 0.5 = 500m). Example: "> 2" triggers violations for containers requesting more than 2 CPU cores. Use this to enforce resource request standards, ensure fair resource allocation, or identify resource-hungry containers.',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'group',
        subComponents: subComponentsForContainerCPU,
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Container CPU limit',
        name: 'Container CPU Limit',
        shortName: 'Container CPU limit',
        description: 'Triggers a violation when the container CPU limit meets the specified comparison criteria (e.g., >, >=, =, <=, <). CPU limits cap the maximum cores a container can use. Example: "< 0.1" triggers violations for containers with CPU limits below 0.1 cores. Use this to enforce that containers set appropriate CPU limits to prevent resource starvation, or to flag containers that could monopolize cluster CPU.',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'group',
        subComponents: subComponentsForContainerCPU,
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Container memory request',
        name: 'Container Memory Request',
        shortName: 'Container memory request',
        description: 'Triggers a violation when the container memory request meets the specified comparison criteria (e.g., >, >=, =, <=, <). Memory requests are in MB. Example: "> 1024" triggers violations for containers requesting more than 1GB of memory. Use this to enforce memory request standards, ensure efficient resource allocation, or identify memory-intensive containers.',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'group',
        subComponents: subComponentsForContainerMemory,
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Container memory limit',
        name: 'Container Memory Limit',
        shortName: 'Container memory limit',
        description: 'Triggers a violation when the container memory limit meets the specified comparison criteria (e.g., >, >=, =, <=, <). Memory limits cap the maximum memory a container can use before being OOMKilled. Measured in MB. Example: "= 0" or no limit set triggers violations. Use this to enforce that all containers have memory limits to prevent OOM issues affecting other workloads.',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'group',
        subComponents: subComponentsForContainerMemory,
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Privileged',
        name: 'Privileged Container',
        shortName: 'Privileged container',
        longName: 'Privileged container status',
        description: 'Triggers a violation when the privileged field in the PodSecurityContext matches the selected value. Selecting "Privileged container" triggers violations for containers running in privileged mode, which gives them nearly all capabilities of the host. Privileged containers can access host devices, modify kernel parameters, and bypass most security restrictions. Use this to block privileged containers in production.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Root filesystem writeability',
        name: 'Read-Only Root Filesystem',
        shortName: 'Root filesystem writeability',
        longName: 'Root filesystem is',
        description: 'Triggers a violation based on the readOnlyRootFilesystem setting in PodSecurityContext. Selecting "Writable" triggers violations for containers with writable root filesystems. Selecting "Read-only" triggers for read-only filesystems. Use "Writable" to enforce read-only root filesystems, improving security by preventing runtime modifications to the container image and limiting attack surface.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Seccomp profile type',
        name: 'Seccomp Profile Type',
        shortName: 'Seccomp profile type',
        longName: 'Seccomp profile type is',
        negatedName: 'Seccomp profile type is not',
        description: 'Triggers a violation when the seccomp (secure computing mode) profile type matches (or doesn\'t match if negated) the specified type. Options are: UNCONFINED (no syscall restrictions), RUNTIME_DEFAULT (container runtime\'s default profile), or LOCALHOST (custom profile). Example: selecting "UNCONFINED" triggers violations for containers without seccomp restrictions. Use this to enforce seccomp profiles that limit available system calls, reducing the kernel attack surface.',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'radioGroupString',
        radioButtons: Object.keys(seccompProfileTypeLabels).map((key) => ({
            text: seccompProfileTypeLabels[key],
            value: key,
        })),
        canBooleanLogic: false,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Privilege escalation',
        name: 'Allow Privilege Escalation',
        shortName: 'Privilege escalation',
        longName: 'Privilege escalation on container is',
        description: 'Triggers a violation based on the allowPrivilegeEscalation setting. Selecting "Allowed" triggers violations when privilege escalation IS allowed. Selecting "Not allowed" triggers when it\'s blocked. Privilege escalation allows a process to gain more privileges than its parent (e.g., via setuid binaries). Use "Allowed" to block privilege escalation and enforce the principle of least privilege.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Share host network namespace',
        name: 'Host Network',
        shortName: 'Host network',
        longName: 'Share host network namespace',
        description: 'Triggers a violation when hostNetwork matches the selected value. Selecting "Uses host network namespace" triggers violations for containers sharing the host\'s network namespace, giving them full access to host network interfaces. This bypasses network isolation and allows seeing all network traffic. Use this to block host network access in production, enforcing proper network isolation.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Share host PID namespace',
        name: 'Host PID',
        shortName: 'Host PID',
        longName: 'Share host PID namespace',
        description: 'Triggers a violation when hostPID matches the selected value. Selecting "Uses host PID namespace" triggers violations for containers sharing the host\'s process ID namespace, allowing them to see and potentially signal all processes on the host. This breaks process isolation. Use this to block host PID access, maintaining process namespace boundaries.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Share host IPC namespace',
        name: 'Host IPC',
        shortName: 'Host IPC',
        longName: 'Share host IPC namespace',
        description: 'Triggers a violation when hostIPC matches the selected value. Selecting "Uses host IPC namespace" triggers violations for containers sharing the host\'s IPC namespace (shared memory, semaphores, message queues). This allows access to host inter-process communication. Use this to block host IPC access, preventing potential information leakage through shared memory.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        name: 'Drop Capabilities',
        shortName: 'Drop capabilities',
        longName: 'Capabilities that MUST be dropped',
        description: 'Triggers a violation when one or more of the specified Linux capabilities are NOT dropped from the container. With AND logic, ALL selected capabilities must be dropped or the policy triggers. For example, selecting "SYS_ADMIN" AND "NET_ADMIN" triggers violations if the container fails to drop either capability. Use this to enforce removal of dangerous capabilities like SYS_ADMIN, reducing the attack surface.',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'select',
        options: [...dropCapabilities],
        canBooleanLogic: false,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        name: 'Add Capabilities',
        shortName: 'Add capabilities',
        longName: 'Capabilities that may NOT be added',
        description: 'Triggers a violation when any of the specified Linux capabilities ARE added to the container. With OR logic, if the container adds ANY of the selected capabilities, the policy triggers. For example, selecting "NET_RAW" OR "SYS_PTRACE" triggers violations if either capability is added. Use this to prevent addition of dangerous capabilities that could enable privilege escalation or system manipulation.',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'select',
        options: [...addCapabilities],
        canBooleanLogic: false,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        name: 'Process Name',
        shortName: 'Process name',
        longName: 'Process name is',
        negatedName: 'Process name doesn\'t match',
        description: 'Triggers a violation at RUNTIME when a process executing in a deployment has a name matching (or not matching if negated) the specified pattern. Regular expressions are supported. Example: specifying "apt-get" triggers violations when apt-get executes. Use this to detect package manager usage, cryptocurrency miners, or other unauthorized executables. Only fires when the process actually runs.',
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'text',
        placeholder: 'apt-get',
        canBooleanLogic: true,
        lifecycleStages: ['RUNTIME'],
    },
    {
        name: 'Process Ancestor',
        shortName: 'Process ancestor',
        longName: 'Process ancestor is',
        negatedName: 'Process ancestor doesn\'t match',
        description: 'Triggers a violation at RUNTIME when a process has a parent/ancestor process matching (or not matching if negated) the specified pattern. Ancestor checks the entire process tree. Regular expressions are supported. Example: specifying "java" triggers violations for any process spawned from a Java process. Use this to detect suspicious process chains or shell spawns from applications.',
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'text',
        placeholder: 'java',
        canBooleanLogic: true,
        lifecycleStages: ['RUNTIME'],
    },
    {
        name: 'Process Arguments',
        shortName: 'Process arguments',
        longName: 'Process arguments are',
        negatedName: 'Process arguments don\'t match',
        description: 'Triggers a violation at RUNTIME when a process is executed with command-line arguments matching (or not matching if negated) the specified pattern. Regular expressions are supported. Example: specifying "install nmap" triggers violations when a package manager attempts to install nmap. Use this to detect specific malicious commands, data exfiltration attempts, or unauthorized tool installations.',
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'text',
        placeholder: 'install nmap',
        canBooleanLogic: true,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Process UID',
        name: 'Process UID',
        shortName: 'Process UID',
        longName: 'Process UID is',
        negatedName: 'Process UID doesn\'t match',
        description: 'Triggers a violation at RUNTIME when a process executes under a user ID (UID) matching (or not matching if negated) the specified pattern. Regular expressions are supported. Example: specifying "^0$" triggers violations for processes running as root (UID 0). Use this to detect privilege escalation or processes running with unexpected user privileges at runtime.',
        category: policyCriteriaCategories.PROCESS_ACTIVITY,
        type: 'text',
        placeholder: '0',
        canBooleanLogic: true,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Network baselining enabled',
        name: 'Unexpected Network Flow Detected',
        shortName: 'Unexpected network flow detected',
        longName: 'Network baselining status',
        description: 'Triggers a violation at RUNTIME when network traffic deviates from the established baseline. Selecting "Unexpected network flow" triggers violations for connections NOT in the locked network baseline. Selecting "Expected network flow" triggers for traffic that IS in the baseline. Use "Unexpected network flow" to detect anomalous network communications, potential data exfiltration, or C2 connections after baselining the deployment.',
        category: policyCriteriaCategories.BASELINE_DEVIATION,
        type: 'radioGroup',
        radioButtons: [
            { text: 'Unexpected network flow', value: true },
            { text: 'Expected network flow', value: false },
        ],
        defaultValue: false,
        reverse: false,
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Process baselining enabled',
        name: 'Unexpected Process Executed',
        shortName: 'Unexpected process executed',
        longName: 'Process baselining status',
        description: 'Triggers a violation at RUNTIME when processes execute that are not in the locked process baseline. Selecting "Unexpected process" triggers violations for processes NOT in the baseline. Selecting "Expected process" triggers for processes that ARE in the baseline. Use "Unexpected process" to detect anomalous executions, malware, or unauthorized tools after establishing a baseline of normal process activity.',
        category: policyCriteriaCategories.BASELINE_DEVIATION,
        type: 'radioGroup',
        radioButtons: [
            { text: 'Unexpected process', value: true },
            { text: 'Expected process', value: false },
        ],
        defaultValue: false,
        reverse: false,
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },

    {
        name: 'Writable Host Mount',
        shortName: 'Host mount writability',
        longName: 'Host mount is',
        description: 'Triggers a violation when a host path (hostPath volume) is mounted with write permissions. Selecting "Writable" triggers violations for writable host mounts. Selecting "Read-only" triggers for read-only mounts. Use "Writable" to block containers from modifying the host filesystem, preventing persistent malware installation, log tampering, or host system compromise.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Service account',
        name: 'Service Account',
        shortName: 'Service account',
        longName: 'Service account name is',
        negatedName: 'Service account name doesn\'t match',
        description: 'Triggers a violation when the Kubernetes service account name matches (or doesn\'t match if negated) the specified pattern. Regular expressions are supported. Example: specifying "^default$" triggers violations for deployments using the "default" service account. Use this to enforce service account naming conventions, prevent use of overly privileged accounts, or require specific service accounts for sensitive workloads.',
        category: policyCriteriaCategories.ACCESS_CONTROL,
        type: 'text',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Automount service account token',
        name: 'Automount Service Account Token',
        shortName: 'Automount service account token',
        description: 'Triggers a violation based on whether the service account token is automatically mounted into the pod. Selecting "Automount" triggers violations when the token IS automatically mounted. Selecting "Do not mount" triggers when it\'s NOT mounted. Use "Automount" to prevent unnecessary API access, reducing the risk of token theft and unauthorized cluster access when the application doesn\'t need to interact with the Kubernetes API.',
        category: policyCriteriaCategories.ACCESS_CONTROL,
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Minimum RBAC permissions',
        name: 'Minimum RBAC Permissions',
        shortName: 'Minimum RBAC permissions',
        longName: 'RBAC permission level is at least',
        negatedName: 'RBAC permission level is less than',
        description: 'Triggers a violation when the deployment\'s service account has RBAC permissions at or above the specified level. Levels are: DEFAULT (minimal), ELEVATED_IN_NAMESPACE (namespace admin), ELEVATED_CLUSTER_WIDE (cross-namespace privileges), CLUSTER_ADMIN (full cluster control). Example: selecting "ELEVATED_CLUSTER_WIDE" triggers violations for deployments with cluster-wide or cluster-admin permissions. Use this to detect overly permissive RBAC grants and enforce least privilege.',
        category: policyCriteriaCategories.ACCESS_CONTROL,
        type: 'select',
        options: Object.keys(rbacPermissionLabels).map((key) => ({
            label: rbacPermissionLabels[key],
            value: key,
        })),
        canBooleanLogic: false,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Required image label',
        name: 'Required Image Label',
        shortName: 'Require image label',
        description: 'Triggers a violation when an image is MISSING a Docker label matching the specified key and optionally value pattern. Regular expressions are supported for both fields. Requires Docker registry integration. Example: key="maintainer", value=".*" triggers violations for images without a maintainer label. Use this to enforce image metadata standards, ensure documentation, or validate build pipeline tags.',
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
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Disallow image label',
        name: 'Disallowed Image Label',
        shortName: 'Disallow image label',
        description: 'Triggers a violation when an image HAS a Docker label matching the specified key and optionally value pattern. Regular expressions are supported for both fields. Requires Docker registry integration. Example: key="internal", value="true" triggers violations for images marked as internal. Use this to block deprecated labels, prevent use of test/debug images in production, or enforce label prohibition policies.',
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
        lifecycleStages: ['BUILD', 'DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Namespace',
        name: 'Namespace',
        shortName: 'Namespace',
        longName: 'Namespace is',
        negatedName: 'Namespace doesn\'t match',
        description: 'Triggers a violation when the Kubernetes namespace name matches (or doesn\'t match if negated) the specified pattern. Regular expressions are supported. Example: specifying "^default$" triggers violations for deployments in the default namespace. Use this to enforce namespace naming conventions, restrict deployments to specific namespaces, or prevent use of default/system namespaces for applications.',
        category: policyCriteriaCategories.DEPLOYMENT_METADATA,
        type: 'text',
        placeholder: 'default',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Container name',
        name: 'Container Name',
        shortName: 'Container name',
        longName: 'Container name is',
        negatedName: 'Container name doesn\'t match',
        description: 'Triggers a violation when a container name within the pod matches (or doesn\'t match if negated) the specified pattern. Regular expressions are supported. Example: specifying ".*sidecar.*" triggers violations for containers with "sidecar" in the name. Use this to enforce container naming conventions, scope policies to specific containers in multi-container pods, or identify containers by naming patterns.',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'text',
        placeholder: 'default',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'AppArmor profile',
        name: 'AppArmor Profile',
        shortName: 'AppArmor profile',
        longName: 'AppArmor profile is',
        negatedName: 'AppArmor profile doesn\'t match',
        description: 'Triggers a violation when the AppArmor profile annotation matches (or doesn\'t match if negated) the specified pattern. AppArmor provides mandatory access control (MAC) to restrict container capabilities. Regular expressions are supported. Example: specifying "unconfined" triggers violations for containers without AppArmor restrictions. Use this to enforce AppArmor profiles, improving container security through kernel-level access controls.',
        category: policyCriteriaCategories.CONTAINER_CONFIGURATION,
        type: 'text',
        placeholder: 'default',
        canBooleanLogic: true,
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Kubernetes action',
        name: 'Kubernetes Resource',
        shortName: 'Kubernetes action',
        description: 'Triggers a violation at RUNTIME when a user performs the specified Kubernetes action on a pod. Actions include: Pod exec (executing commands in containers via kubectl exec), or Pods port forward (forwarding local ports to pod ports). Use this to detect interactive access to containers, which may indicate troubleshooting, administrative actions, or potential unauthorized access attempts.',
        category: policyCriteriaCategories.USER_ISSUED_CONTAINER_COMMANDS,
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
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Kubernetes user name',
        name: 'Kubernetes User Name',
        shortName: 'Kubernetes user name',
        negatedName: 'Kubernetes user name doesn\'t match',
        description: 'Triggers a violation at RUNTIME when the Kubernetes user performing pod exec or port forward actions matches (or doesn\'t match if negated) the specified pattern. Regular expressions are supported. Example: specifying "admin.*" triggers violations when any admin user executes commands in pods. Use this with Kubernetes action criteria to restrict who can perform interactive operations.',
        category: policyCriteriaCategories.USER_ISSUED_CONTAINER_COMMANDS,
        type: 'text',
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Kubernetes user groups',
        name: 'Kubernetes User Groups',
        shortName: 'Kubernetes user groups',
        negatedName: 'Kubernetes user group doesn\'t match',
        description: 'Triggers a violation at RUNTIME when the Kubernetes user\'s group performing pod exec or port forward actions matches (or doesn\'t match if negated) the specified pattern. Regular expressions are supported. Example: specifying "system:.*" triggers violations for system groups. Use this with Kubernetes action criteria to detect or restrict interactive access by group membership.',
        category: policyCriteriaCategories.USER_ISSUED_CONTAINER_COMMANDS,
        type: 'text',
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Replicas',
        name: 'Replicas',
        shortName: 'Replicas',
        description: 'Triggers a violation when the number of deployment replicas meets the specified comparison criteria (e.g., >, >=, =, <=, <). Example: "= 1" triggers violations for single-replica deployments (no high availability). "< 2" triggers for deployments with fewer than 2 replicas. Use this to enforce high availability requirements or detect deployments that may not scale properly. Note: The admission controller blocks scale operations that would violate the policy.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Liveness probe',
        name: 'Liveness Probe Defined',
        shortName: 'Liveness probe',
        longName: 'Liveness probe is',
        description: 'Triggers a violation based on whether a liveness probe is defined. Selecting "Not defined" triggers violations for containers WITHOUT liveness probes. Selecting "Defined" triggers when a probe EXISTS. Liveness probes detect unresponsive containers so Kubernetes can restart them. Use "Not defined" to enforce liveness probes, improving application reliability and automatic recovery from failures.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Readiness probe',
        name: 'Readiness Probe Defined',
        shortName: 'Readiness probe',
        longName: 'Readiness probe is',
        description: 'Triggers a violation based on whether a readiness probe is defined. Selecting "Not defined" triggers violations for containers WITHOUT readiness probes. Selecting "Defined" triggers when a probe EXISTS. Readiness probes determine when a container is ready to accept traffic. Use "Not defined" to enforce readiness probes, ensuring only healthy containers receive traffic and preventing premature load balancing.',
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
        lifecycleStages: ['DEPLOY', 'RUNTIME'],
    },
    {
        label: 'Mounted file path',
        name: 'Mounted File Path',
        shortName: 'Mounted file path',
        category: policyCriteriaCategories.FILE_ACTIVITY,
        type: 'select',
        placeholder: 'Select an option',
        options: fileActivityPathOptions,
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
        featureFlagDependency: ['ROX_SENSITIVE_FILE_ACTIVITY'],
    },
    {
        label: 'File operation',
        name: 'File Operation',
        shortName: 'File operation',
        category: policyCriteriaCategories.FILE_ACTIVITY,
        type: 'select',
        placeholder: 'Select an option',
        options: fileOperationOptions,
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
        featureFlagDependency: ['ROX_SENSITIVE_FILE_ACTIVITY'],
    },
];

export const auditLogDescriptor: Descriptor[] = [
    {
        label: 'Kubernetes API verb',
        name: 'Kubernetes API Verb',
        shortName: 'Kubernetes API verb',
        description: 'Triggers a violation at RUNTIME when the specified Kubernetes API verb is used in audit log events. Verbs are: CREATE, DELETE, GET, PATCH, UPDATE. Example: selecting "DELETE" triggers violations for deletion operations. Use this with resource type criteria to detect sensitive operations like secret deletion or role modification. Requires audit log event source.',
        category: policyCriteriaCategories.RESOURCE_OPERATION,
        type: 'select',
        placeholder: 'Select an API verb',
        options: APIVerbs,
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Kubernetes resource type',
        name: 'Kubernetes Resource',
        shortName: 'Kubernetes resource type',
        description: 'Triggers a violation at RUNTIME when the specified Kubernetes resource type is accessed in audit log events. Resources include: ConfigMaps, Secrets, ClusterRoles, ClusterRoleBindings, NetworkPolicies, SecurityContextConstraints, EgressFirewalls. Example: selecting "SECRETS" triggers violations for secret operations. Use this with API verb criteria to monitor sensitive resource access. Requires audit log event source.',
        category: policyCriteriaCategories.RESOURCE_OPERATION,
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
            {
                label: 'ClusterRoles',
                value: 'CLUSTER_ROLES',
            },
            {
                label: 'ClusterRoleBindings',
                value: 'CLUSTER_ROLE_BINDINGS',
            },
            {
                label: 'NetworkPolicies',
                value: 'NETWORK_POLICIES',
            },
            {
                label: 'SecurityContextConstraints',
                value: 'SECURITY_CONTEXT_CONSTRAINTS',
            },
            {
                label: 'EgressFirewalls',
                value: 'EGRESS_FIREWALLS',
            },
        ],
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Kubernetes resource name',
        name: 'Kubernetes Resource Name',
        shortName: 'Kubernetes resource name',
        negatedName: 'Kubernetes resource name doesn\'t match',
        description: 'Triggers a violation at RUNTIME when the name of the accessed Kubernetes resource in audit logs matches (or doesn\'t match if negated) the specified pattern. Regular expressions are supported. Example: "prod-.*" triggers violations for resources with names starting with "prod-". Use this to monitor access to specific sensitive resources. Requires audit log event source.',
        category: policyCriteriaCategories.RESOURCE_ATTRIBUTES,
        type: 'text',
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Kubernetes user name',
        name: 'Kubernetes User Name',
        shortName: 'Kubernetes user name',
        negatedName: 'Kubernetes user name doesn\'t match',
        description: 'Triggers a violation at RUNTIME when the user accessing resources in audit logs matches (or doesn\'t match if negated) the specified pattern. Regular expressions are supported. Example: "system:serviceaccount:.*" triggers violations for service account access. Use this to detect unauthorized user access to sensitive resources. Requires audit log event source.',
        category: policyCriteriaCategories.RESOURCE_ATTRIBUTES,
        type: 'text',
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Kubernetes user group',
        name: 'Kubernetes User Groups',
        shortName: 'Kubernetes user groups',
        negatedName: 'Kubernetes user group doesn\'t match',
        description: 'Triggers a violation at RUNTIME when the user\'s group accessing resources in audit logs matches (or doesn\'t match if negated) the specified pattern. Regular expressions are supported. Example: "system:masters" triggers violations for cluster-admin group access. Use this to monitor privileged group activity. Requires audit log event source.',
        category: policyCriteriaCategories.RESOURCE_ATTRIBUTES,
        type: 'text',
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'User agent',
        name: 'User Agent',
        shortName: 'User agent',
        longName: 'User agent is',
        negatedName: 'User agent doesn\'t match',
        description: 'Triggers a violation at RUNTIME when the user agent accessing resources in audit logs matches (or doesn\'t match if negated) the specified pattern. User agent identifies the client tool (e.g., kubectl, oc, curl). Regular expressions are supported. Example: "kubectl.*" triggers violations for kubectl access. Use this to detect unusual client tools or automated access patterns. Requires audit log event source.',
        category: policyCriteriaCategories.RESOURCE_ATTRIBUTES,
        type: 'text',
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Source IP address',
        name: 'Source IP Address',
        shortName: 'Source IP address',
        longName: 'Source IP address is',
        negatedName: 'Source IP address doesn\'t match',
        description: 'Triggers a violation at RUNTIME when the source IP address in audit logs matches (or doesn\'t match if negated) the specified pattern. Supports IPv4 and IPv6. Regular expressions are supported. Example: "10\\.0\\..*" triggers violations for access from the 10.0.0.0/16 network. Use this to detect access from unexpected locations or enforce IP allowlisting. Requires audit log event source.',
        category: policyCriteriaCategories.RESOURCE_ATTRIBUTES,
        type: 'text',
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'Is impersonated user',
        name: 'Is Impersonated User',
        shortName: 'Is impersonated user',
        description: 'Triggers a violation at RUNTIME based on whether the request in audit logs uses user impersonation. Selecting "True" triggers violations when a user or service account is impersonating another identity. Selecting "False" triggers for non-impersonated requests. User impersonation allows acting as another user. Use "True" to monitor impersonation usage, which may indicate privilege escalation or testing. Requires audit log event source.',
        category: policyCriteriaCategories.RESOURCE_ATTRIBUTES,
        type: 'radioGroup',
        radioButtons: [
            { text: 'True', value: true },
            { text: 'False', value: false },
        ],
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
];

export const nodeEventDescriptor: Descriptor[] = [
    {
        label: 'Node file path',
        name: 'Node File Path',
        shortName: 'Node file path',
        category: policyCriteriaCategories.FILE_ACTIVITY,
        type: 'select',
        placeholder: 'Select a file path',
        options: fileActivityPathOptions,
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
    {
        label: 'File operation',
        name: 'File Operation',
        shortName: 'File operation',
        category: policyCriteriaCategories.FILE_ACTIVITY,
        type: 'select',
        placeholder: 'Select an option',
        options: fileOperationOptions,
        canBooleanLogic: false,
        lifecycleStages: ['RUNTIME'],
    },
];
