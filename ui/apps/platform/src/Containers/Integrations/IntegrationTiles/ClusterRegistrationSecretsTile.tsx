import React, { ReactElement } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import { fetchClusterRegistrationSecrets } from 'services/ClustersService';
import { clustersClusterRegistrationSecretsPath } from 'routePaths';

import { clusterRegistrationSecretDescriptor as descriptor } from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';

const { image, label } = descriptor;

function ClusterRegistrationSecretsTile(): ReactElement {
    const { data } = useRestQuery(fetchClusterRegistrationSecrets);
    const numIntegrations = data?.items.length ?? 0;

    return (
        <IntegrationTile
            image={image}
            label={label}
            linkTo={clustersClusterRegistrationSecretsPath}
            numIntegrations={numIntegrations}
        />
    );
}

export default ClusterRegistrationSecretsTile;
