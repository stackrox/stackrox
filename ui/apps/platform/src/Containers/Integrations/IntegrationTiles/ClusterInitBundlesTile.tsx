import React, { ReactElement, useEffect, useState } from 'react';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { fetchClusterInitBundles } from 'services/ClustersService';
import { clustersInitBundlesPath } from 'routePaths';

import {
    authenticationTokensSource as source,
    clusterInitBundleDescriptor as descriptor,
    getIntegrationsListPath,
} from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';

const { image, label, type } = descriptor;

function ClusterInitBundlesTile(): ReactElement {
    const [numIntegrations, setNumIntegrations] = useState(0);
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isMoveInitBundlesEnabled = isFeatureFlagEnabled('ROX_MOVE_INIT_BUNDLES_UI');

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
            linkTo={
                isMoveInitBundlesEnabled
                    ? clustersInitBundlesPath
                    : getIntegrationsListPath(source, type)
            }
            numIntegrations={numIntegrations}
        />
    );
}

export default ClusterInitBundlesTile;
