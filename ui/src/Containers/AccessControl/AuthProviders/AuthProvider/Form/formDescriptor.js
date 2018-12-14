import React from 'react';

const formDescriptors = {
    oidc: [
        {
            label: 'Integration Name',
            jsonPath: 'name',
            type: 'text',
            placeholder: 'Auth0'
        },
        {
            label: 'Callback Mode',
            jsonPath: 'config.mode',
            type: 'select',
            options: [
                { value: 'fragment', label: 'Fragment' },
                { value: 'post', label: 'HTTP POST' }
            ],
            default: 'fragment',
            immutable: true
        },
        {
            label: 'Issuer',
            jsonPath: 'config.issuer',
            type: 'text',
            placeholder: 'your-tenant.auth0.com',
            immutable: true
        },
        {
            label: 'Client ID',
            jsonPath: 'config.client_id',
            type: 'text',
            placeholder: '',
            immutable: true
        }
    ],
    auth0: [
        {
            label: 'Integration Name',
            jsonPath: 'name',
            type: 'text',
            placeholder: 'Auth0'
        },
        {
            label: 'Auth0 Tenant',
            jsonPath: 'config.issuer',
            type: 'text',
            placeholder: 'your-tenant.auth0.com',
            immutable: true
        },
        {
            label: 'Client ID',
            jsonPath: 'config.client_id',
            type: 'text',
            placeholder: '',
            immutable: true
        }
    ],
    saml: [
        {
            label: 'Integration Name',
            jsonPath: 'name',
            type: 'text',
            placeholder: 'Shibboleth'
        },
        {
            label: 'ServiceProvider Issuer',
            jsonPath: 'config.sp_issuer',
            type: 'text',
            placeholder: 'https://prevent.stackrox.io/',
            immutable: true
        },
        {
            html: (
                <div className="border-b border-base-400 border-dotted flex pb-2">
                    Option 1: Dynamic Configuration
                </div>
            ),
            type: 'html'
        },
        {
            label: 'IdP Metadata URL',
            jsonPath: 'config.idp_metadata_url',
            type: 'text',
            placeholder: 'https://idp.example.com/metadata',
            immutable: true
        },
        {
            html: (
                <div className="border-b border-base-400 border-dotted flex pb-2">
                    Option 2: Static Configuration
                </div>
            ),
            type: 'html'
        },
        {
            label: 'IdP Issuer',
            jsonPath: 'config.idp_issuer',
            type: 'text',
            placeholder: 'https://idp.example.com/',
            immutable: true
        },
        {
            label: 'IdP SSO URL',
            jsonPath: 'config.idp_sso_url',
            type: 'text',
            placeholder: 'https://idp.example.com/login',
            immutable: true
        },
        {
            label: 'IdP Certificate (PEM)',
            jsonPath: 'config.idp_cert_pem',
            type: 'textarea',
            placeholder:
                '-----BEGIN CERTIFICATE-----\nYour certificate data\n-----END CERTIFICATE-----',
            immutable: true
        }
    ],
    attrToRole: {
        keyOptions: [
            { value: 'name', label: 'name' },
            { value: 'email', label: 'email' },
            { value: 'uid', label: 'uid' },
            { value: 'groups', label: 'groups' }
        ],
        roleOptions: [
            { value: 'admin', label: 'Admin' },
            { value: 'analyst', label: 'Analyst' },
            { value: 'none', label: 'None' },
            { value: 'continuous', label: 'Continuous' },
            { value: 'sensor creator', label: 'Sensor Creator' }
        ]
    }
};

export default formDescriptors;
