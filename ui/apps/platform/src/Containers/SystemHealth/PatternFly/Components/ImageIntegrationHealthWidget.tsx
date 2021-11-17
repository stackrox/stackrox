import React, { useState, useEffect, ReactElement } from 'react';

import { fetchImageIntegrationsHealth } from 'services/IntegrationHealthService';
import { fetchImageIntegrations } from 'services/ImageIntegrationsService';
import integrationsList from 'Containers/Integrations/utils/integrationsList';
import IntegrationHealthWidgetVisual from './IntegrationHealthWidgetVisual';
import { mergeIntegrationResponses, IntegrationMergedItem } from '../utils/integrations';

type WidgetProps = {
    pollingCount: number;
};

const ImageIntegrationHealthWidget = ({ pollingCount }: WidgetProps): ReactElement => {
    const [imageIntegrationsMerged, setImageIntegrationsMerged] = useState(
        [] as IntegrationMergedItem[]
    );
    const [imageIntegrationsRequestHasError, setImageIntegrationsRequestHasError] = useState(false);

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
                setImageIntegrationsRequestHasError(false);
            })
            .catch(() => {
                setImageIntegrationsMerged([]);
                setImageIntegrationsRequestHasError(true);
            });
    }, [pollingCount]);

    return (
        <IntegrationHealthWidgetVisual
            id="image-integrations"
            integrationText="Image Integrations"
            integrationsMerged={imageIntegrationsMerged}
            requestHasError={imageIntegrationsRequestHasError}
        />
    );
};

export default ImageIntegrationHealthWidget;
