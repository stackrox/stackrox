import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { selectors } from 'reducers';

import {
    getIntegrationsListPath,
    imageIntegrationsDescriptors as descriptors,
    imageIntegrationsSource as source,
} from '../utils/integrationsList';
import IntegrationsSection from './IntegrationsSection';
import IntegrationTile from './IntegrationTile';
import { featureFlagDependencyFilterer, integrationTypeCounter } from './integrationTiles.utils';

function ImageIntegrationsSection(): ReactElement {
    const integrations = useSelector(selectors.getImageIntegrations);
    const countIntegrations = integrationTypeCounter(integrations);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const featureFlagDependencyFilter = featureFlagDependencyFilterer(isFeatureFlagEnabled);

    return (
        <IntegrationsSection headerName="Image Integrations" id="image-integrations">
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
                    />
                );
            })}
        </IntegrationsSection>
    );
}

export default ImageIntegrationsSection;
