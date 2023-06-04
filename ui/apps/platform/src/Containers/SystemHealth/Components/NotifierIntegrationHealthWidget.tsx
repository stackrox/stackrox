import React, { useState, useEffect, ReactElement } from 'react';

import { fetchPluginIntegrationsHealth } from 'services/IntegrationHealthService';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import integrationsList from 'Containers/Integrations/utils/integrationsList';
import IntegrationHealthWidgetVisual from './IntegrationHealthWidgetVisual';
import { mergeIntegrationResponses, IntegrationMergedItem } from '../utils/integrations';
import { getAxiosErrorMessage } from '../../../utils/responseErrorUtils';

type WidgetProps = {
    pollingCount: number;
};

const NotifierIntegrationHealthWidget = ({ pollingCount }: WidgetProps): ReactElement => {
    const [notifiersMerged, setNotifiersMerged] = useState([] as IntegrationMergedItem[]);
    const [errorMessageFetching, setErrorMessageFetching] = useState('');

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
                setErrorMessageFetching('');
            })
            .catch((error) => {
                setNotifiersMerged([]);
                setErrorMessageFetching(getAxiosErrorMessage(error));
            });
    }, [pollingCount]);

    return (
        <IntegrationHealthWidgetVisual
            integrationText="Notifier Integrations"
            integrationsMerged={notifiersMerged}
            errorMessageFetching={errorMessageFetching}
        />
    );
};

export default NotifierIntegrationHealthWidget;
