import type { ReactElement } from 'react';
import { selectors } from 'reducers';
import { useSelector } from 'react-redux';
import IntegrationTile from './IntegrationTile';
import {
    authenticationTokensSource as source,
    getIntegrationsListPath,
    machineAccessDescriptor as descriptor,
} from '../utils/integrationsList';

function MachineAccessTile(): ReactElement {
    const { image, label, type } = descriptor;
    const integrations = useSelector(selectors.getMachineAccessConfigs);
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
