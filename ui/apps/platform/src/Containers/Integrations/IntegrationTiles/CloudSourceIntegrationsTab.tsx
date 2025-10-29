import type { ReactElement } from 'react';
import { Alert, Gallery } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { fetchCloudSources } from 'services/CloudSourceService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import IntegrationsTabPage from './IntegrationsTabPage';
import type { IntegrationsTabProps } from './IntegrationsTab.types';

import PaladinCloudTile from './PaladinCloudTile';
import OcmTile from './OcmTile';

const source = 'cloudSources';

function CloudSourceIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {
    const { data, error } = useRestQuery(fetchCloudSources);
    const integrations = data?.response?.cloudSources ?? [];

    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            {error && (
                <Alert variant="warning" title="Unable to get integratons" isInline component="p">
                    {getAxiosErrorMessage(error)}
                </Alert>
            )}
            <Gallery hasGutter>
                <PaladinCloudTile integrations={integrations} />
                <OcmTile integrations={integrations} />
            </Gallery>
        </IntegrationsTabPage>
    );
}

export default CloudSourceIntegrationsTab;
