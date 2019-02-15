import React from 'react';

const baseURL = `${window.location.protocol}//${window.location.host}`;
const oidcFragmentCallbackURL = `${baseURL}/auth/response/oidc`;
const oidcPostCallbackURL = `${baseURL}/sso/providers/oidc/callback`;
const samlACSURL = `${baseURL}/sso/providers/saml/acs`;

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
        },
        {
            html: (
                <div className="text-tertiary-800 bg-tertiary-200 p-3 pb-2 rounded border-2 border-tertiary-300 ">
                    <p className="border-b-2 border-tertiary-300 pb-3">
                        <strong>Note: </strong> if required by your IdP, use the following callback
                        URLs:
                    </p>
                    <ul className="pl-4 mt-2 leading-loose">
                        <li>
                            For <span className="font-700">Fragment</span> mode:{' '}
                            <a
                                className="text-tertiary-800 hover:text-tertiary-900"
                                href={oidcFragmentCallbackURL}
                            >
                                {oidcFragmentCallbackURL}
                            </a>
                        </li>
                        <li>
                            For <span className="font-700">HTTP POST</span> mode:{' '}
                            <a
                                className="text-tertiary-800 hover:text-tertiary-900"
                                href={oidcPostCallbackURL}
                            >
                                {oidcPostCallbackURL}
                            </a>
                        </li>
                    </ul>
                </div>
            ),
            type: 'html'
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
        },
        {
            html: (
                <div className="text-tertiary-800 bg-tertiary-200 p-3 pb-2 rounded border-2 border-tertiary-300 ">
                    <p className="border-b-2 border-tertiary-300 pb-3">
                        <strong>Note: </strong> if required by your IdP, use the following callback
                        URL:
                    </p>
                    <ul className="pl-4 mt-2 leading-loose">
                        <li>
                            <a
                                className="text-tertiary-800 hover:text-tertiary-900"
                                href={oidcFragmentCallbackURL}
                            >
                                {oidcFragmentCallbackURL}
                            </a>
                        </li>
                    </ul>
                </div>
            ),
            type: 'html'
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
        },
        {
            html: (
                <div className="text-tertiary-800 bg-tertiary-200 p-3 pb-2 rounded border-2 border-tertiary-300 ">
                    <p className="border-b-2 border-tertiary-300 pb-3">
                        <strong>Note: </strong> if required by your IdP, use the following Assertion
                        Consumer Service (ACS) URL:
                    </p>
                    <ul className="pl-4 mt-2 leading-loose">
                        <li>
                            <a
                                className="text-tertiary-800 hover:text-tertiary-900"
                                href={samlACSURL}
                            >
                                {samlACSURL}
                            </a>
                        </li>
                    </ul>
                </div>
            ),
            type: 'html'
        }
    ],
    attrToRole: {
        keyOptions: [
            { value: 'name', label: 'name' },
            { value: 'email', label: 'email' },
            { value: 'uid', label: 'uid' },
            { value: 'groups', label: 'groups' }
        ]
    }
};

export default formDescriptors;
