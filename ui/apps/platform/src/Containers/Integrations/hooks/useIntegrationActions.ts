import { useHistory } from 'react-router-dom';
import { integrationsPath } from 'routePaths';

import {
    createIntegration,
    saveIntegrationV2,
    testIntegrationV2,
} from 'services/IntegrationsService';
import { IntegrationSource, IntegrationType } from 'Containers/Integrations/utils/integrationUtils';
import useFetchIntegrations from './useFetchIntegrations';
import usePageState from './usePageState';

export type FormResponseMessage = {
    message: string;
    isError: boolean;
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
            if (isEditing) {
                await saveIntegrationV2(source, data);
            } else {
                await createIntegration(source, data);
            }
            fetchIntegrations();
            history.push(integrationsListPath);
            return null;
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
