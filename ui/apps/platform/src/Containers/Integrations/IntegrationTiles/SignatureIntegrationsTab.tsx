import type { ReactElement } from 'react';
import { Alert, Gallery } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { fetchSignatureIntegrations } from 'services/SignatureIntegrationsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import type { IntegrationsTabProps } from './IntegrationsTab.types';
import IntegrationsTabPage from './IntegrationsTabPage';
import IntegrationTile from './IntegrationTile';
import {
    getIntegrationsListPath,
    signatureIntegrationDescriptor as descriptor,
    signatureIntegrationsSource as source,
} from '../utils/integrationsList';

const { image, label, type } = descriptor;

function SignatureIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {
    const { data, error } = useRestQuery(fetchSignatureIntegrations);
    const integrations = data ?? [];

    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            {error && (
                <Alert variant="warning" title="Unable to get integratons" isInline component="p">
                    {getAxiosErrorMessage(error)}
                </Alert>
            )}
            <Gallery hasGutter>
                <IntegrationTile
                    image={image}
                    label={label}
                    linkTo={getIntegrationsListPath(source, type)}
                    numIntegrations={integrations.length}
                />
            </Gallery>
        </IntegrationsTabPage>
    );
}

export default SignatureIntegrationsTab;
