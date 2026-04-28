import type { ReactElement } from 'react';
import { Gallery } from '@patternfly/react-core';

import IntegrationsTabPage from './IntegrationsTabPage';
import type { IntegrationsTabProps } from './IntegrationsTab.types';
import ExternalIntegrationTile from './ExternalIntegrationTile';

import { apiClientDescriptors } from '../utils/integrationsList';

const source = 'apiClients';

function ApiClientsIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {
    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            <Gallery hasGutter>
                {apiClientDescriptors.map(({ Logo, label, externalUrl }) => (
                    <ExternalIntegrationTile
                        key={label}
                        Logo={Logo}
                        label={label}
                        url={externalUrl}
                    />
                ))}
            </Gallery>
        </IntegrationsTabPage>
    );
}

export default ApiClientsIntegrationsTab;
