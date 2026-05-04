import type { ReactElement } from 'react';
import { Gallery } from '@patternfly/react-core';

import IntegrationsTabPage from './IntegrationsTabPage';
import type { IntegrationsTabProps } from './IntegrationsTab.types';

import ServiceNowTile from './ServiceNowTile';

const source = 'apiClients';

function ApiClientsIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {
    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            <Gallery hasGutter>
                <ServiceNowTile integrations={[]} />
            </Gallery>
        </IntegrationsTabPage>
    );
}

export default ApiClientsIntegrationsTab;
