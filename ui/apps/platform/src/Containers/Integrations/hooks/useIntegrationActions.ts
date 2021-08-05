import { useHistory } from 'react-router-dom';
import { integrationsPath } from 'routePaths';

import {
    createIntegration,
    saveIntegrationV2,
    testIntegrationV2,
} from 'services/IntegrationsService';
import { IntegrationSource, IntegrationType } from 'Containers/Integrations/utils/integrationUtils';
import { generateAPIToken } from 'services/APITokensService';
import { generateClusterInitBundle } from 'services/ClustersService';
import useFetchIntegrations from './useFetchIntegrations';
import usePageState from './usePageState';

export type FormResponseMessage = {
    message: string;
    isError: boolean;
    responseData?: unknown;
} | null;

export type UseIntegrationActions = {
    source: IntegrationSource;
    type: IntegrationType;
};

export type UseIntegrationActionsResult = {
    onSave: (data) => Promise<FormResponseMessage>;
    onTest: (data) => Promise<FormResponseMessage>;
    onCancel: () => void;
};

function useIntegrationActions(): UseIntegrationActionsResult {
    const history = useHistory();
    const {
        isEditing,
        params: { source, type },
    } = usePageState();
    const fetchIntegrations = useFetchIntegrations(source);
    const integrationsListPath = `${integrationsPath}/${source}/${type}`;

    async function onSave(data) {
        try {
            let responseData;
            if (isEditing) {
                responseData = await saveIntegrationV2(source, data);
            } else if (type === 'apitoken') {
                responseData = await generateAPIToken(data);
            } else if (type === 'clusterInitBundle') {
                responseData = await generateClusterInitBundle(data);
            } else {
                responseData = await createIntegration(source, data);
            }
            fetchIntegrations();
            return { message: 'Integration was saved successfully', isError: false, responseData };
        } catch (error) {
            return { message: error?.response?.data?.error || error, isError: true };
        }
    }

    async function onTest(data) {
        try {
            await testIntegrationV2(source, data);
            return { message: 'Test was successful', isError: false };
        } catch (error) {
            return { message: error?.response?.data?.error || error, isError: true };
        }
    }

    function onCancel() {
        history.push(integrationsListPath);
    }

    return { onSave, onTest, onCancel };
}

export default useIntegrationActions;
