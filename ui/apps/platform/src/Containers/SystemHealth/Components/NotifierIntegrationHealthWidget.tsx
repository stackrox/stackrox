import React, { useState, useEffect, ReactElement } from 'react';

import { fetchPluginIntegrationsHealth } from 'services/IntegrationHealthService';
import { fetchIntegration } from 'services/IntegrationsService';
import integrationsList from 'Containers/Integrations/integrationsList';
import IntegrationHealthWidgetVisual from './IntegrationHealthWidgetVisual';
import { mergeIntegrationResponses, IntegrationMergedItem } from '../utils/integrations';

type WidgetProps = {
    pollingCount: number;
};

const NotifierIntegrationHealthWidget = ({ pollingCount }: WidgetProps): ReactElement => {
    const [notifiersMerged, setNotifiersMerged] = useState([] as IntegrationMergedItem[]);
    const [notifiersRequestHasError, setNotifiersRequestHasError] = useState(false);

    useEffect(() => {
        Promise.all([fetchPluginIntegrationsHealth(), fetchIntegration('notifiers')])
            .then(([integrationsHealth, { response }]) => {
                setNotifiersMerged(
                    mergeIntegrationResponses(
                        integrationsHealth,
                        response.notifiers,
                        integrationsList.notifiers
                    )
                );
                setNotifiersRequestHasError(false);
            })
            .catch(() => {
                setNotifiersMerged([]);
                setNotifiersRequestHasError(true);
            });
    }, [pollingCount]);

    return (
        <IntegrationHealthWidgetVisual
            id="notifier-integrations"
            integrationText="Notifier Integrations"
            integrationsMerged={notifiersMerged}
            requestHasError={notifiersRequestHasError}
        />
    );
};

export default NotifierIntegrationHealthWidget;
