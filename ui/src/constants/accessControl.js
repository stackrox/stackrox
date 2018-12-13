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
