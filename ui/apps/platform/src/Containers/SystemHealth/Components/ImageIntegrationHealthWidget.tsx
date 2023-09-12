import React, { ReactElement, useEffect, useState } from 'react';

import { fetchImageIntegrationsHealth } from 'services/IntegrationHealthService';
import { fetchImageIntegrations } from 'services/ImageIntegrationsService';
import { imageIntegrationsDescriptors } from 'Containers/Integrations/utils/integrationsList';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import IntegrationHealthWidgetVisual from './IntegrationHealthWidgetVisual';
import { IntegrationMergedItem, mergeIntegrationResponses } from '../utils/integrations';

type WidgetProps = {
    pollingCount: number;
};

const ImageIntegrationHealthWidget = ({ pollingCount }: WidgetProps): ReactElement => {
    const [isFetching, setIsFetching] = useState(false);
    const [imageIntegrationsMerged, setImageIntegrationsMerged] = useState(
        [] as IntegrationMergedItem[]
    );
    const [errorMessageFetching, setErrorMessageFetching] = useState('');

    useEffect(() => {
        setIsFetching(true);
        Promise.all([fetchImageIntegrationsHealth(), fetchImageIntegrations()])
            .then(([integrationsHealth, integrations]) => {
                setImageIntegrationsMerged(
                    mergeIntegrationResponses(
                        integrationsHealth,
                        integrations,
                        imageIntegrationsDescriptors
                    )
                );
                setErrorMessageFetching('');
            })
            .catch((error) => {
                setImageIntegrationsMerged([]);
                setErrorMessageFetching(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsFetching(false);
            });
    }, [pollingCount]);
    const isFetchingInitialRequest = isFetching && pollingCount === 0;

    return (
        <IntegrationHealthWidgetVisual
            integrationText="Image Integrations"
            integrationsMerged={imageIntegrationsMerged}
            errorMessageFetching={errorMessageFetching}
            isFetchingInitialRequest={isFetchingInitialRequest}
        />
    );
};

export default ImageIntegrationHealthWidget;
