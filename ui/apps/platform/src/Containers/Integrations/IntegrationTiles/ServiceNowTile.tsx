import type { ReactElement } from 'react';

import {
    apiClientsSource as source,
    getIntegrationsListPath,
    serviceNowDescriptor as descriptor,
} from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';
import { integrationTypeCounter } from './integrationTiles.utils';

const { Logo, label, type } = descriptor;

export type ServiceNowTileProps = {
    integrations: { type: string }[];
};

function ServiceNowTile({ integrations }: ServiceNowTileProps): ReactElement {
    const countIntegrations = integrationTypeCounter(integrations);

    return (
        <IntegrationTile
            Logo={Logo}
            label={label}
            linkTo={getIntegrationsListPath(source, type)}
            numIntegrations={countIntegrations('serviceNow')}
        />
    );
}

export default ServiceNowTile;
