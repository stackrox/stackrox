/* constants specific to Roles */
export const NO_ACCESS = 'NO_ACCESS';
export const READ_ACCESS = 'READ_ACCESS';
export const READ_WRITE_ACCESS = 'READ_WRITE_ACCESS';

export type AccessLevel = 'NO_ACCESS' | 'READ_ACCESS' | 'READ_WRITE_ACCESS';

const defaultRoles = {
    Admin: true,
    Analyst: true,
    'Continuous Integration': true,
    None: true,
    'Scope Manager': true,
    'Sensor Creator': true,
    'Vulnerability Management Approver': true,
    'Vulnerability Management Requester': true,
    'Vulnerability Report Creator': true,
};

export function getIsDefaultRoleName(name: string): boolean {
    return Boolean(defaultRoles[name]);
}

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
    'Alert',
    'Cluster',
    'Config',
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
