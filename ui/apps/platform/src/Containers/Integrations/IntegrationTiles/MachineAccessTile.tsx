import React, { ReactElement, useEffect, useState } from 'react';
import { fetchMachineAccessConfigs } from 'services/MachineAccessService';
import IntegrationTile from './IntegrationTile';
import {
    authenticationTokensSource as source,
    machineAccessDescriptor as descriptor,
    getIntegrationsListPath,
} from '../utils/integrationsList';

function MachineAccessConfigTile(): ReactElement {
    const { image, label, type } = descriptor;

    const [numIntegrations, setNumIntegrations] = useState(0);

    useEffect(() => {
        fetchMachineAccessConfigs()
            .then(({ response: { configs } }) => {
                setNumIntegrations(configs.length);
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

export default MachineAccessConfigTile;
