import React, { ReactElement } from 'react';

import useAuthStatus from 'hooks/useAuthStatus';

import APITokensTile from './APITokensTile';
import ClusterInitBundles from './ClusterInitBundlesTile';
import IntegrationsSection from './IntegrationsSection';

function AuthenticationTokensSection(): ReactElement {
    // TODO after 4.4 release:
    // Delete ClusterInitBundles tile from integrations.
    // Delete unreachable code.
    // Delete request from integration sagas and data from integrations reducer.
    const { currentUser } = useAuthStatus();
    const hasAdminRole = Boolean(currentUser?.userInfo?.roles.some(({ name }) => name === 'Admin')); // optional chaining just in case of the unexpected

    return (
        <IntegrationsSection headerName="Authentication Tokens" id="token-integrations">
            <APITokensTile />
            {hasAdminRole && <ClusterInitBundles />}
        </IntegrationsSection>
    );
}

export default AuthenticationTokensSection;
