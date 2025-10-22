import type { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import {
    paladinCloudDescriptor as descriptor,
    cloudSourcesSource as source,
    getIntegrationsListPath,
} from '../utils/integrationsList';
import { selectors } from '../../../reducers';
import IntegrationTile from './IntegrationTile';
import { integrationTypeCounter } from './integrationTiles.utils';

const { image, label, type } = descriptor;

function PaladinCloudTile(): ReactElement {
    const integrations = useSelector(selectors.getCloudSources);
    const countIntegrations = integrationTypeCounter(integrations);

    return (
        <IntegrationTile
            image={image}
            label={label}
            linkTo={getIntegrationsListPath(source, type)}
            numIntegrations={countIntegrations('TYPE_PALADIN_CLOUD')}
        />
    );
}

export default PaladinCloudTile;
