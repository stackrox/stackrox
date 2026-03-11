import type { ReactElement } from 'react';

import type { CloudSourceIntegration } from 'services/CloudSourceService';

import {
    cloudSourcesSource as source,
    getIntegrationsListPath,
    paladinCloudDescriptor as descriptor,
} from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';
import { integrationTypeCounter } from './integrationTiles.utils';

const { ImageComponent, label, type } = descriptor;

export type PaladinCloudTileProps = {
    integrations: CloudSourceIntegration[];
};

function PaladinCloudTile({ integrations }: PaladinCloudTileProps): ReactElement {
    const countIntegrations = integrationTypeCounter(integrations);

    return (
        <IntegrationTile
            ImageComponent={ImageComponent}
            label={label}
            linkTo={getIntegrationsListPath(source, type)}
            numIntegrations={countIntegrations('TYPE_PALADIN_CLOUD')}
        />
    );
}

export default PaladinCloudTile;
