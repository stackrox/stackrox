import React, { ReactElement } from 'react';

import useAuthStatus from 'hooks/useAuthStatus';
import usePermissions from 'hooks/usePermissions';

import APITokensTile from './APITokensTile';
import ClusterInitBundles from './ClusterInitBundlesTile';
import ClusterRegistrationSecrets from './ClusterRegistrationSecretsTile';
import IntegrationsSection from './IntegrationsSection';
import MachineAccessTile from './MachineAccessTile';

function AuthenticationTokensSection(): ReactElement {
    // TODO after 4.4 release:
    // Delete ClusterInitBundles tile from integrations.
    // Delete unreachable code.
    // Delete request from integration sagas and data from integrations reducer.
    const { currentUser } = useAuthStatus();
    const { hasReadAccess } = usePermissions();
    const hasAdminRole = Boolean(currentUser?.userInfo?.roles.some(({ name }) => name === 'Admin')); // optional chaining just in case of the unexpected
    const hasReadAccessForAccess = hasReadAccess('Access');

    return (
        <IntegrationsSection headerName="Authentication Tokens" id="token-integrations">
            <APITokensTile />
            {hasAdminRole && <ClusterInitBundles />}
            {hasAdminRole && <ClusterRegistrationSecrets />}
            {hasReadAccessForAccess && <MachineAccessTile />}
        </IntegrationsSection>
    );
}

export default AuthenticationTokensSection;
