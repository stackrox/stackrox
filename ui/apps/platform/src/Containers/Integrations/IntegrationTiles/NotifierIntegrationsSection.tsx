import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { selectors } from 'reducers';

import {
    getIntegrationsListPath,
    notifierIntegrationsDescriptors as descriptors,
    notifierIntegrationsSource as source,
} from '../utils/integrationsList';
import IntegrationsSection from './IntegrationsSection';
import IntegrationTile from './IntegrationTile';
import { featureFlagDependencyFilterer, integrationTypeCounter } from './integrationTiles.utils';

function NotifierIntegrationsSection(): ReactElement {
    const integrations = useSelector(selectors.getNotifiers);
    const countIntegrations = integrationTypeCounter(integrations);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const featureFlagDependencyFilter = featureFlagDependencyFilterer(isFeatureFlagEnabled);

    return (
        <IntegrationsSection headerName="Notifier Integrations" id="notifier-integrations">
            {descriptors.filter(featureFlagDependencyFilter).map((descriptor) => {
                const { image, label, type } = descriptor;

                return (
                    <IntegrationTile
                        key={type}
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

export default NotifierIntegrationsSection;
