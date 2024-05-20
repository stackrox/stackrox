import React, { ReactElement, useEffect, useState } from 'react';

import { fetchClusterInitBundles } from 'services/ClustersService';
import { clustersInitBundlesPath } from 'routePaths';

import { clusterInitBundleDescriptor as descriptor } from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';

const { image, label } = descriptor;

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
            linkTo={clustersInitBundlesPath}
            numIntegrations={numIntegrations}
        />
    );
}

export default ClusterInitBundlesTile;
