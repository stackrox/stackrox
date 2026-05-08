import { useNavigate } from 'react-router-dom-v5-compat';
import { integrationsPath } from 'routePaths';

import {
    createIntegration,
    saveIntegration,
    saveIntegrationV2,
    testIntegration,
    testIntegrationV2,
} from 'services/IntegrationsService';
import type {
    IntegrationOptions,
    IntegrationSource as CrudIntegrationSource,
} from 'services/IntegrationsService';
import { generateAPIToken } from 'services/APITokensService';
import { getAxiosErrorMessage, isTimeoutError } from 'utils/responseErrorUtils';

import type { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { createMachineAccessConfig } from 'services/MachineAccessService';

import type { IntegrationSource, IntegrationType } from '../utils/integrationUtils';
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
    const navigate = useNavigate();
    const {
        isEditing,
        params: { source, type },
    } = usePageState();
    // The routing IntegrationSource type includes 'apiClients', which has no backend
    // endpoints. This hook only mounts on create/edit routes, so 'apiClients' never
    // reaches here. Safe to narrow to the CRUD-capable source type.
    const crudSource = source as CrudIntegrationSource;
    const fetchIntegrations = useFetchIntegrations(source);
    const integrationsListPath = `${integrationsPath}/${source}/${type}`;

    async function onSave(data, { updatePassword }: IntegrationOptions = {}) {
        try {
            let responseData;

            if (isEditing) {
                responseData =
                    typeof updatePassword === 'boolean'
                        ? await saveIntegration(crudSource, data, { updatePassword })
                        : await saveIntegrationV2(crudSource, data);
                navigate(integrationsListPath);
            } else if (type === 'apitoken') {
                responseData = await generateAPIToken(data);
            } else if (type === 'machineAccess') {
                responseData = await createMachineAccessConfig(data);
                navigate(-1);
            } else {
                responseData = await createIntegration(crudSource, data);
                // we only want to redirect when creating a new non-apitoken integration
                navigate(-1);
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
                await testIntegration(crudSource, data, { updatePassword });
            } else {
                await testIntegrationV2(crudSource, data);
            }
            return { message: `The test was successful`, isError: false };
        } catch (error) {
            if (source === 'cloudSources' && isTimeoutError(error)) {
                return { message: 'Could not reach the cloud source endpoint.', isError: true };
            }
            return { message: getAxiosErrorMessage(error), isError: true };
        }
    }

    function onCancel() {
        navigate(integrationsListPath);
    }

    return { onSave, onTest, onCancel };
}

export default useIntegrationActions;
