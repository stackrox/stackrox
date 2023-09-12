import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { selectors } from 'reducers';

import {
    backupIntegrationsDescriptors as descriptors,
    backupIntegrationsSource as source,
    getIntegrationsListPath,
} from '../utils/integrationsList';
import IntegrationTile from './IntegrationTile';
import IntegrationsSection from './IntegrationsSection';
import { featureFlagDependencyFilterer, integrationTypeCounter } from './integrationTiles.utils';

function BackupIntegrationsSection(): ReactElement {
    const integrations = useSelector(selectors.getBackups);
    const countIntegrations = integrationTypeCounter(integrations);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const featureFlagDependencyFilter = featureFlagDependencyFilterer(isFeatureFlagEnabled);

    return (
        <IntegrationsSection headerName="Backup Integrations" id="backup-integrations">
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

export default BackupIntegrationsSection;
