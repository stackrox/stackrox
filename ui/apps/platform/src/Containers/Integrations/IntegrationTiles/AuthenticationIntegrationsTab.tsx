import type { ReactElement } from 'react';
import { Gallery } from '@patternfly/react-core';

import usePermissions from 'hooks/usePermissions';

import type { IntegrationsTabProps } from './IntegrationsTab.types';
import IntegrationsTabPage from './IntegrationsTabPage';

import APITokensTile from './APITokensTile';
import MachineAccessTile from './MachineAccessTile';

const source = 'authProviders';

function AuthenticationIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {
    // Delete request from integration sagas and data from integrations reducer.
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForAccess = hasReadAccess('Access');

    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            <Gallery hasGutter>
                <APITokensTile />
                {hasReadAccessForAccess && <MachineAccessTile />}
            </Gallery>
        </IntegrationsTabPage>
    );
}

export default AuthenticationIntegrationsTab;
