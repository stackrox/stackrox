import React, { useState, useEffect, ReactElement } from 'react';

import { fetchLogIntegrationsHealth } from 'services/IntegrationHealthService';
import { fetchIntegration } from 'services/IntegrationsService';
import integrationsList from 'Containers/Integrations/integrationsList';
import IntegrationHealthWidgetVisual from './IntegrationHealthWidgetVisual';
import { mergeIntegrationResponses, IntegrationMergedItem } from '../utils/integrations';

type WidgetProps = {
    pollingCount: number;
};

const LogIntegrationHealthWidget = ({ pollingCount }: WidgetProps): ReactElement => {
    const [logIntegrationsMerged, setLogIntegrationsMerged] = useState(
        [] as IntegrationMergedItem[]
    );
    const [logIntegrationsRequestHasError, setLogIntegrationsRequestHasError] = useState(false);

    useEffect(() => {
        Promise.all([fetchLogIntegrationsHealth(), fetchIntegration('logIntegrations')])
            .then(([integrationsHealth, { response }]) => {
                setLogIntegrationsMerged(
                    mergeIntegrationResponses(
                        integrationsHealth,
                        response.integrations,
                        integrationsList.logIntegrations
                    )
                );
                setLogIntegrationsRequestHasError(false);
            })
            .catch(() => {
                setLogIntegrationsMerged([]);
                setLogIntegrationsRequestHasError(true);
            });
    }, [pollingCount]);

    return (
        <IntegrationHealthWidgetVisual
            id="log-integrations"
            integrationText="Audit Logging Integrations"
            integrationsMerged={logIntegrationsMerged}
            requestHasError={logIntegrationsRequestHasError}
        />
    );
};

export default LogIntegrationHealthWidget;
