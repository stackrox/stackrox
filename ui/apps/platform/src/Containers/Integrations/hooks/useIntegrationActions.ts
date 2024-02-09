import { useHistory } from 'react-router-dom';
import { integrationsPath } from 'routePaths';

import {
    IntegrationOptions,
    createIntegration,
    saveIntegration,
    saveIntegrationV2,
    testIntegration,
    testIntegrationV2,
} from 'services/IntegrationsService';
import { IntegrationSource, IntegrationType } from 'Containers/Integrations/utils/integrationUtils';
import { generateAPIToken } from 'services/APITokensService';
import { generateClusterInitBundle } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { createMachineAccessConfig } from 'services/MachineAccessService';
import useFetchIntegrations from './useFetchIntegrations';
import usePageState from './usePageState';

export type UseIntegrationActions = {
    source: IntegrationSource;
    type: IntegrationType;
};

export type UseIntegrationActionsResult = {
    onSave: (data, options?: IntegrationOptions) => Promise<FormResponseMessage>;
    onTest: (data, options?: IntegrationOptions) => Promise<FormResponseMessage>;
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

    async function onSave(data, { updatePassword }: IntegrationOptions = {}) {
        try {
            let responseData;

            if (isEditing) {
                responseData =
                    typeof updatePassword === 'boolean'
                        ? await saveIntegration(source, data, { updatePassword })
                        : await saveIntegrationV2(source, data);
                history.push(integrationsListPath);
            } else if (type === 'apitoken') {
                responseData = await generateAPIToken(data);
            } else if (type === 'clusterInitBundle') {
                responseData = await generateClusterInitBundle(data);
            } else if (type === 'machineAccess') {
                responseData = await createMachineAccessConfig(data);
                history.goBack();
            } else {
                responseData = await createIntegration(source, data);
                // we only want to redirect when creating a new (non-apitoken and non-clusterinitbundle) integration
                history.goBack();
            }

            fetchIntegrations();
            return { message: 'Integration was saved successfully', isError: false, responseData };
        } catch (error) {
            return { message: getAxiosErrorMessage(error), isError: true };
        }
    }

    async function onTest(data, { updatePassword }: IntegrationOptions = {}) {
        try {
            if (typeof updatePassword === 'boolean') {
                await testIntegration(source, data, { updatePassword });
            } else {
                await testIntegrationV2(source, data);
            }
            return { message: `The test was successful`, isError: false };
        } catch (error) {
            return { message: getAxiosErrorMessage(error), isError: true };
        }
    }

    function onCancel() {
        history.push(integrationsListPath);
    }

    return { onSave, onTest, onCancel };
}

export default useIntegrationActions;
