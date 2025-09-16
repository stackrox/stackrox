import React, { ReactElement } from 'react';

import usePermissions from 'hooks/usePermissions';

import APITokensTile from './APITokensTile';
import IntegrationsSection from './IntegrationsSection';
import MachineAccessTile from './MachineAccessTile';

function AuthenticationTokensSection(): ReactElement {
    // Delete request from integration sagas and data from integrations reducer.
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForAccess = hasReadAccess('Access');

    return (
        <IntegrationsSection headerName="Authentication Tokens" id="token-integrations">
            <APITokensTile />
            {hasReadAccessForAccess && <MachineAccessTile />}
        </IntegrationsSection>
    );
}

export default AuthenticationTokensSection;
