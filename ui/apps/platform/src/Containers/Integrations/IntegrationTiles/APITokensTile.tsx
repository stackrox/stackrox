import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';

import { selectors } from 'reducers';

import {
    apiTokenDescriptor as descriptor,
    authenticationTokensSource as source,
    getIntegrationsListPath,
} from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';

const { image, label, type } = descriptor;

function APITokensTile(): ReactElement {
    const integrations = useSelector(selectors.getAPITokens);

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
