import type { ReactElement } from 'react';

import type { AiIntegration } from 'services/AiIntegrationsService';

import {
    aiIntegrationsSource as source,
    getIntegrationsListPath,
    lightspeedDescriptor as descriptor,
} from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';
import { integrationTypeCounter } from './integrationTiles.utils';

const { Logo, label, type } = descriptor;

export type LightspeedTileProps = {
    integrations: AiIntegration[];
};

function LightspeedTile({ integrations }: LightspeedTileProps): ReactElement {
    const countIntegrations = integrationTypeCounter(integrations);

    return (
        <IntegrationTile
            Logo={Logo}
            label={label}
            linkTo={getIntegrationsListPath(source, type)}
            numIntegrations={countIntegrations('AI_INTEGRATION_TYPE_OLS')}
        />
    );
}

export default LightspeedTile;
