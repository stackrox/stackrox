import React, { ReactElement, useEffect, useState } from 'react';

import { fetchClusterRegistrationSecrets } from 'services/ClustersService';
import { clustersClusterRegistrationSecretsPath } from 'routePaths';

import { clusterRegistrationSecretDescriptor as descriptor } from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';

const { image, label } = descriptor;

function ClusterRegistrationSecretsTile(): ReactElement {
    const [numIntegrations, setNumIntegrations] = useState(0);

    useEffect(() => {
        fetchClusterRegistrationSecrets()
            .then(({ response: { items } }) => {
                setNumIntegrations(items.length);
            })
            .catch(() => {});
    }, []);

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
