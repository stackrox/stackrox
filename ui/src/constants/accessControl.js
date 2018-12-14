/* constants specific to Roles */
export const NO_ACCESS = 'NO_ACCESS';
export const READ_ACCESS = 'READ_ACCESS';
export const READ_WRITE_ACCESS = 'READ_WRITE_ACCESS';

export const defaultRoles = {
    Admin: true,
    Analyst: true,
    'Continuous Integration': true,
    None: true,
    'Sensor Creator': true
};

/* constants specific to Auth Providers */
export const availableAuthProviders = [
    {
        label: 'Auth0',
        value: 'auth0'
    },
    {
        label: 'OpenID Connect',
        value: 'oidc'
    },
    {
        label: 'SAML 2.0',
        value: 'saml'
    }
];

export const defaultPermissions = {
    APIToken: NO_ACCESS,
    Alert: NO_ACCESS,
    AuthProvider: NO_ACCESS,
    Benchmark: NO_ACCESS,
    BenchmarkScan: NO_ACCESS,
    BenchmarkSchedule: NO_ACCESS,
    BenchmarkTrigger: NO_ACCESS,
    Cluster: NO_ACCESS,
    DebugMetrics: NO_ACCESS,
    Deployment: NO_ACCESS,
    Detection: NO_ACCESS,
    Group: NO_ACCESS,
    Image: NO_ACCESS,
    ImageIntegration: NO_ACCESS,
    ImbuedLogs: NO_ACCESS,
    Indicator: NO_ACCESS,
    Node: NO_ACCESS,
    Notifier: NO_ACCESS,
    NetworkPolicy: NO_ACCESS,
    NetworkGraph: NO_ACCESS,
    Policy: NO_ACCESS,
    Role: NO_ACCESS,
    Secret: NO_ACCESS,
    ServiceIdentity: NO_ACCESS,
    User: NO_ACCESS
};
