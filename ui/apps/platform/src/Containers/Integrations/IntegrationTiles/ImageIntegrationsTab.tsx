import type { ReactElement } from 'react';
import { Alert, Gallery } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useRestQuery from 'hooks/useRestQuery';
import { fetchImageIntegrations } from 'services/ImageIntegrationsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import {
    getIntegrationsListPath,
    imageIntegrationsDescriptors as descriptors,
    imageIntegrationsSource as source,
} from '../utils/integrationsList';
import type { IntegrationsTabProps } from './IntegrationsTab.types';
import IntegrationsTabPage from './IntegrationsTabPage';
import IntegrationTile from './IntegrationTile';
import { featureFlagDependencyFilterer, integrationTypeCounter } from './integrationTiles.utils';

function ImageIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {
    const { data, error } = useRestQuery(fetchImageIntegrations);
    const integrations = data ?? [];
    const countIntegrations = integrationTypeCounter(integrations);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const featureFlagDependencyFilter = featureFlagDependencyFilterer(isFeatureFlagEnabled);

    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            {error && (
                <Alert variant="warning" title="Unable to get integratons" isInline component="p">
                    {getAxiosErrorMessage(error)}
                </Alert>
            )}
            <Gallery hasGutter>
                {descriptors.filter(featureFlagDependencyFilter).map((descriptor) => {
                    const { categories, image, label, type } = descriptor;

                    return (
                        <IntegrationTile
                            key={type}
                            categories={categories}
                            image={image}
                            label={label}
                            linkTo={getIntegrationsListPath(source, type)}
                            numIntegrations={countIntegrations(type)}
                            isTechPreview={false}
                        />
                    );
                })}
            </Gallery>
        </IntegrationsTabPage>
    );
}

export default ImageIntegrationsTab;
