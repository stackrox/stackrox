import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import {
    paladinCloudDescriptor as descriptor,
    cloudSourcesSource as source,
    getIntegrationsListPath,
} from '../utils/integrationsList';
import { selectors } from '../../../reducers';
import IntegrationTile from './IntegrationTile';

const { image, label, type } = descriptor;

function PaladinCloudTile(): ReactElement {
    const integrations = useSelector(selectors.getCloudSources);

    return (
        <IntegrationTile
            image={image}
            label={label}
            linkTo={getIntegrationsListPath(source, type)}
            numIntegrations={integrations.length}
        />
    );
}

export default PaladinCloudTile;
