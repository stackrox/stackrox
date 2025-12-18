import type { ReactElement } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import { fetchMachineAccessConfigs } from 'services/MachineAccessService';

import {
    authenticationTokensSource as source,
    getIntegrationsListPath,
    machineAccessDescriptor as descriptor,
} from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';

const { image, label, type } = descriptor;

function MachineAccessTile(): ReactElement {
    const { data } = useRestQuery(fetchMachineAccessConfigs);
    const integrations = data?.response?.configs ?? [];

    return (
        <IntegrationTile
            image={image}
            label={label}
            linkTo={getIntegrationsListPath(source, type)}
            numIntegrations={integrations.length}
        />
    );
}

export default MachineAccessTile;
