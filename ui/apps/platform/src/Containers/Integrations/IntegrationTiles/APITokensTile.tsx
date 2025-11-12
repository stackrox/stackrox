import type { ReactElement } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import { fetchAPITokens } from 'services/APITokensService';

import {
    apiTokenDescriptor as descriptor,
    authenticationTokensSource as source,
    getIntegrationsListPath,
} from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';

const { image, label, type } = descriptor;

function APITokensTile(): ReactElement {
    const { data } = useRestQuery(fetchAPITokens);
    const integrations = data?.response?.tokens ?? [];

    return (
        <IntegrationTile
            image={image}
            label={label}
            linkTo={getIntegrationsListPath(source, type)}
            numIntegrations={integrations.length}
        />
    );
}

export default APITokensTile;
