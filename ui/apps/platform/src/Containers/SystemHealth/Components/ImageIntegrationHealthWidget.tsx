import React, { useState, useEffect, ReactElement } from 'react';

import { fetchImageIntegrationsHealth } from 'services/IntegrationHealthService';
import { fetchImageIntegrations } from 'services/ImageIntegrationsService';
import integrationsList from 'Containers/Integrations/utils/integrationsList';
import IntegrationHealthWidgetVisual from './IntegrationHealthWidgetVisual';
import { mergeIntegrationResponses, IntegrationMergedItem } from '../utils/integrations';
import { getAxiosErrorMessage } from '../../../utils/responseErrorUtils';

type WidgetProps = {
    pollingCount: number;
};

const ImageIntegrationHealthWidget = ({ pollingCount }: WidgetProps): ReactElement => {
    const [imageIntegrationsMerged, setImageIntegrationsMerged] = useState(
        [] as IntegrationMergedItem[]
    );
    const [errorMessageFetching, setErrorMessageFetching] = useState('');

    useEffect(() => {
        Promise.all([fetchImageIntegrationsHealth(), fetchImageIntegrations()])
            .then(([integrationsHealth, integrations]) => {
                setImageIntegrationsMerged(
                    mergeIntegrationResponses(
                        integrationsHealth,
                        integrations,
                        integrationsList.imageIntegrations
                    )
                );
                setErrorMessageFetching('');
            })
            .catch((error) => {
                setImageIntegrationsMerged([]);
                setErrorMessageFetching(getAxiosErrorMessage(error));
            });
    }, [pollingCount]);

    return (
        <IntegrationHealthWidgetVisual
            integrationText="Image Integrations"
            integrationsMerged={imageIntegrationsMerged}
            errorMessageFetching={errorMessageFetching}
        />
    );
};

export default ImageIntegrationHealthWidget;
