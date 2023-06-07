/* constants specific to Roles */
import { ResourceName } from '../types/roleResources';

export const NO_ACCESS = 'NO_ACCESS';
export const READ_ACCESS = 'READ_ACCESS';
export const READ_WRITE_ACCESS = 'READ_WRITE_ACCESS';

export type AccessLevel = 'NO_ACCESS' | 'READ_ACCESS' | 'READ_WRITE_ACCESS';

export const authProviderLabels = {
    auth0: 'Auth0',
    oidc: 'OpenID Connect',
    saml: 'SAML 2.0',
    userpki: 'User Certificates',
    iap: 'Google IAP',
    openshift: 'OpenShift Auth',
};

export const oidcCallbackModes = [
    {
        label: 'Auto-select (recommended)',
        value: 'auto',
    },
    {
        label: 'HTTP POST',
        value: 'post',
    },
    {
        label: 'Fragment',
        value: 'fragment',
    },
    {
        label: 'Query',
        value: 'query',
    },
];

// DEPRECATED, replaced by map for SAC above
export const oidcCallbackValues = {
    auto: 'Auto-select (recommended)',
    post: 'HTTP POST',
    fragment: 'Fragment',
    query: 'Query',
};

export const defaultMinimalReadAccessResources = [
    'Administration',
    'Alert',
    'Cluster',
    'Deployment',
    'Image',
    'Namespace',
    'NetworkPolicy',
    'NetworkGraph',
    'Node',
    'Policy',
    'Secret',
];

// Default to giving new roles read access to specific resources.
export const defaultNewRolePermissions = defaultMinimalReadAccessResources.reduce(
    (map: Record<string, AccessLevel>, resource) => {
        const newMap = map;
        newMap[resource] = READ_ACCESS;
        return newMap;
    },
    {}
);

export const defaultSelectedRole = {
    name: '',
    resourceToAccess: defaultNewRolePermissions,
};

export const resourceSubstitutions: Record<string, string[]> = {
    Access: ['AuthProvider', 'Group', 'Licenses', 'Role', 'User'],
    Administration: [
        'AllComments',
        'Config',
        'DebugLogs',
        'NetworkGraphConfig',
        'ProbeUpload',
        'ScannerBundle',
        'ScannerDefinitions',
        'SensorUpgradeConfig',
        'ServiceIdentity',
    ],
    Cluster: ['ClusterCVE'],
    Compliance: ['ComplianceRuns'],
    DeploymentExtension: ['Indicator', 'NetworkBaseline', 'ProcessWhitelist', 'Risk'],
    Integration: [
        'APIToken',
        'BackupPlugins',
        'ImageIntegration',
        'Notifier',
        'SignatureIntegration',
    ],
    Image: ['ImageComponent'],
};

export const resourceRemovalReleaseVersions = new Map<ResourceName, string>([]);

// TODO(ROX-11453): Remove this mapping once the old resources are fully deprecated.
export const replacedResourceMapping = new Map<ResourceName, string>([]);

export const deprecatedResourceRowStyle = { backgroundColor: 'rgb(255,250,205)' };
