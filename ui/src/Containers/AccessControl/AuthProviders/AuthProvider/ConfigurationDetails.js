import React from 'react';
import Labeled from 'Components/Labeled';

import { knownBackendFlags as featureFlags } from 'utils/featureFlags';
import FeatureEnabled from 'Containers/FeatureEnabled';

const CommonDetails = ({ name }) => (
    <>
        <Labeled label="Integration Name">{name}</Labeled>
    </>
);

const OidcDetails = ({ authProvider: { name, config } }) => {
    const oidcCallbackValues = {
        post: 'HTTP POST',
        fragment: 'Fragment'
    };
    const callbackModeValue = oidcCallbackValues[config.mode];
    if (!callbackModeValue) throw new Error(`Unknown callback mode "${config.mode}"`);

    return (
        <>
            <CommonDetails name={name} />
            <Labeled label="Callback Mode">{callbackModeValue}</Labeled>
            <Labeled label="Issuer">{config.issuer}</Labeled>
            <Labeled label="Client ID">{config.client_id}</Labeled>
            <FeatureEnabled featureFlag={featureFlags.ROX_REFRESH_TOKENS}>
                <Labeled label="Client Secret">{config.client_secret ? '*****' : null}</Labeled>
            </FeatureEnabled>
        </>
    );
};

const Auth0Details = ({ authProvider: { name, config } }) => (
    <>
        <CommonDetails name={name} />
        <Labeled label="Auth0 Tenant">{config.issuer}</Labeled>
        <Labeled label="Client ID">{config.client_id}</Labeled>
    </>
);

const SamlDetails = ({ authProvider: { name, config } }) => {
    const idpDetails = config.idp_metadata_url ? (
        <Labeled label="Dynamically configured using IdP metadata URL">
            {config.idp_metadata_url}
        </Labeled>
    ) : (
        <>
            <Labeled label="IdP Issuer">{config.idp_issuer}</Labeled>
            <Labeled label="IdP SSO URL">{config.idp_sso_url}</Labeled>
            <Labeled label="Name/ID Format">{config.idp_nameid_format}</Labeled>
            <Labeled label="IdP Certificate (PEM)">
                <pre className="font-500 whitespace-pre-line">{config.idp_cert_pem}</pre>
            </Labeled>
        </>
    );
    return (
        <>
            <CommonDetails name={name} />
            <Labeled label="ServiceProvider Issuer">{config.sp_issuer}</Labeled>
            {idpDetails}
        </>
    );
};

const UserPkiDetails = ({ authProvider: { name, config } }) => (
    <>
        <CommonDetails name={name} />
        <Labeled label="CA Certificates (PEM)">
            <pre className="font-500 whitespace-pre-line">{config.idp_cert_pem}</pre>
        </Labeled>
    </>
);

const detailsComponents = {
    oidc: OidcDetails,
    auth0: Auth0Details,
    saml: SamlDetails,
    userpki: UserPkiDetails
};

const ConfigurationDetails = ({ authProvider }) => {
    const DetailsComponent = detailsComponents[authProvider.type];
    if (!DetailsComponent) throw new Error(`Unknown auth provider type: ${authProvider}`);

    return <DetailsComponent authProvider={authProvider} />;
};

export default ConfigurationDetails;
