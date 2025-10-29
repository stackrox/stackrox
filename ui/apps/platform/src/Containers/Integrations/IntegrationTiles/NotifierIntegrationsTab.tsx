import type { ReactElement } from 'react';
import { Alert, Gallery } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useCentralCapabilities from 'hooks/useCentralCapabilities';
import useRestQuery from 'hooks/useRestQuery';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import {
    getIntegrationsListPath,
    notifierIntegrationsDescriptors as descriptors,
    notifierIntegrationsSource as source,
} from '../utils/integrationsList';
import type { IntegrationsTabProps } from './IntegrationsTab.types';
import IntegrationsTabPage from './IntegrationsTabPage';
import IntegrationTile from './IntegrationTile';
import { featureFlagDependencyFilterer, integrationTypeCounter } from './integrationTiles.utils';

function NotifierIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {
    const { data, error } = useRestQuery(fetchNotifierIntegrations);
    const integrations = data ?? [];
    const countIntegrations = integrationTypeCounter(integrations);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const featureFlagDependencyFilter = featureFlagDependencyFilterer(isFeatureFlagEnabled);

    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const canUseAcscsEmailIntegration = isCentralCapabilityAvailable(
        'centralCanUseAcscsEmailIntegration'
    );

    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            {error && (
                <Alert variant="warning" title="Unable to get integratons" isInline component="p">
                    {getAxiosErrorMessage(error)}
                </Alert>
            )}
            <Gallery hasGutter>
                {descriptors.filter(featureFlagDependencyFilter).map((descriptor) => {
                    const { image, label, type } = descriptor;
                    if (!canUseAcscsEmailIntegration && type === 'acscsEmail') {
                        return null; // TODO add centralCapabilityRequirement to descriptor
                    }

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
            </Gallery>
        </IntegrationsTabPage>
    );
}

export default NotifierIntegrationsTab;
