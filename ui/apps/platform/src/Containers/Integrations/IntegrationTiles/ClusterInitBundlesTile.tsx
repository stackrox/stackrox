import React, { ReactElement, useEffect, useState } from 'react';

import { fetchClusterInitBundles } from 'services/ClustersService';

import {
    authenticationTokensSource as source,
    clusterInitBundleDescriptor as descriptor,
    getIntegrationsListPath,
} from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';

const { image, label, type } = descriptor;

function ClusterInitBundlesTile(): ReactElement {
    const [numIntegrations, setNumIntegrations] = useState(0);

    useEffect(() => {
        fetchClusterInitBundles()
            .then(({ response: { items } }) => {
                setNumIntegrations(items.length);
            })
            .catch(() => {});
    }, []);

    return (
        <IntegrationTile
            image={image}
            label={label}
            linkTo={getIntegrationsListPath(source, type)}
            numIntegrations={numIntegrations}
        />
    );
}

export default ClusterInitBundlesTile;
