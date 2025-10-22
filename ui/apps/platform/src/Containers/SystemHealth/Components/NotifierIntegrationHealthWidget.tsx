import React, { useState, useEffect } from 'react';
import type { ReactElement } from 'react';

import { fetchNotifierIntegrationsHealth } from 'services/IntegrationHealthService';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import { notifierIntegrationsDescriptors } from 'Containers/Integrations/utils/integrationsList';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import IntegrationHealthWidgetVisual from './IntegrationHealthWidgetVisual';
import { mergeIntegrationResponses } from '../utils/integrations';
import type { IntegrationMergedItem } from '../utils/integrations';

type WidgetProps = {
    pollingCount: number;
};

const NotifierIntegrationHealthWidget = ({ pollingCount }: WidgetProps): ReactElement => {
    const [isFetching, setIsFetching] = useState(false);
    const [notifiersMerged, setNotifiersMerged] = useState([] as IntegrationMergedItem[]);
    const [errorMessageFetching, setErrorMessageFetching] = useState('');

    useEffect(() => {
        setIsFetching(true);
        Promise.all([fetchNotifierIntegrationsHealth(), fetchNotifierIntegrations()])
            .then(([integrationsHealth, notifiers]) => {
                setNotifiersMerged(
                    mergeIntegrationResponses(
                        integrationsHealth,
                        notifiers,
                        notifierIntegrationsDescriptors
                    )
                );
                setErrorMessageFetching('');
            })
            .catch((error) => {
                setNotifiersMerged([]);
                setErrorMessageFetching(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsFetching(false);
            });
    }, [pollingCount]);
    const isFetchingInitialRequest = isFetching && pollingCount === 0;

    return (
        <IntegrationHealthWidgetVisual
            integrationText="Notifier Integrations"
            integrationsMerged={notifiersMerged}
            errorMessageFetching={errorMessageFetching}
            isFetchingInitialRequest={isFetchingInitialRequest}
        />
    );
};

export default NotifierIntegrationHealthWidget;
