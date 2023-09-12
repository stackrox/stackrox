import React, { ReactElement } from 'react';

import usePermissions from 'hooks/usePermissions';

import APITokensTile from './APITokensTile';
import ClusterInitBundles from './ClusterInitBundlesTile';
import IntegrationsSection from './IntegrationsSection';

function AuthenticationTokensSection(): ReactElement {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForAdministration = hasReadAccess('Administration');

    return (
        <IntegrationsSection headerName="Authentication Tokens" id="token-integrations">
            <APITokensTile />
            {hasReadAccessForAdministration && <ClusterInitBundles />}
        </IntegrationsSection>
    );
}

export default AuthenticationTokensSection;
