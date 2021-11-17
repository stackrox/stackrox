import React, { useState, useEffect, ReactElement } from 'react';

import { fetchPluginIntegrationsHealth } from 'services/IntegrationHealthService';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import integrationsList from 'Containers/Integrations/utils/integrationsList';
import IntegrationHealthWidgetVisual from './IntegrationHealthWidgetVisual';
import { mergeIntegrationResponses, IntegrationMergedItem } from '../utils/integrations';

type WidgetProps = {
    pollingCount: number;
};

const NotifierIntegrationHealthWidget = ({ pollingCount }: WidgetProps): ReactElement => {
    const [notifiersMerged, setNotifiersMerged] = useState([] as IntegrationMergedItem[]);
    const [notifiersRequestHasError, setNotifiersRequestHasError] = useState(false);

    useEffect(() => {
        Promise.all([fetchPluginIntegrationsHealth(), fetchNotifierIntegrations()])
            .then(([integrationsHealth, notifiers]) => {
                setNotifiersMerged(
                    mergeIntegrationResponses(
                        integrationsHealth,
                        notifiers,
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
