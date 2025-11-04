import type { ReactElement } from 'react';
import { Alert, Gallery } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useRestQuery from 'hooks/useRestQuery';
import { fetchBackupIntegrations } from 'services/BackupIntegrationsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import {
    backupIntegrationsDescriptors as descriptors,
    backupIntegrationsSource as source,
    getIntegrationsListPath,
} from '../utils/integrationsList';
import type { IntegrationsTabProps } from './IntegrationsTab.types';
import IntegrationsTabPage from './IntegrationsTabPage';
import IntegrationTile from './IntegrationTile';
import { featureFlagDependencyFilterer, integrationTypeCounter } from './integrationTiles.utils';

function BackupIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {
    const { data, error } = useRestQuery(fetchBackupIntegrations);
    const integrations = data ?? [];
    const countIntegrations = integrationTypeCounter(integrations);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const featureFlagDependencyFilter = featureFlagDependencyFilterer(isFeatureFlagEnabled);

    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            {error && (
                <Alert variant="danger" title="Unable to get integrations" isInline component="p">
                    {getAxiosErrorMessage(error)}
                </Alert>
            )}
            <Gallery hasGutter>
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
            </Gallery>
        </IntegrationsTabPage>
    );
}

export default BackupIntegrationsTab;
