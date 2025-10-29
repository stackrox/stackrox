import type { ReactElement } from 'react';

import type { CloudSourceIntegration } from 'services/CloudSourceService';

import {
    cloudSourcesSource as source,
    getIntegrationsListPath,
    ocmDescriptor as descriptor,
} from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';
import { integrationTypeCounter } from './integrationTiles.utils';

const { image, label, type } = descriptor;

export type OcmTileProps = {
    integrations: CloudSourceIntegration[];
};

function OcmTile({ integrations }: OcmTileProps): ReactElement {
    const countIntegrations = integrationTypeCounter(integrations);

    return (
        <IntegrationTile
            image={image}
            label={label}
            linkTo={getIntegrationsListPath(source, type)}
            numIntegrations={countIntegrations('TYPE_OCM')}
        />
    );
}

export default OcmTile;
