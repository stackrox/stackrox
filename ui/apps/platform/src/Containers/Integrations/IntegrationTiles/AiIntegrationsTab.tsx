import type { ReactElement } from 'react';
import { Alert, Gallery } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { fetchAiIntegrations } from 'services/AiIntegrationsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import IntegrationsTabPage from './IntegrationsTabPage';
import type { IntegrationsTabProps } from './IntegrationsTab.types';

import LightspeedTile from './LightspeedTile';

const source = 'aiIntegrations';

function AiIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {
    const { data, error } = useRestQuery(fetchAiIntegrations);
    const integrations = data ?? [];

    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            {error && (
                <Alert variant="danger" title="Unable to get integrations" isInline component="p">
                    {getAxiosErrorMessage(error)}
                </Alert>
            )}
            <Gallery hasGutter>
                <LightspeedTile integrations={integrations} />
            </Gallery>
        </IntegrationsTabPage>
    );
}

export default AiIntegrationsTab;
