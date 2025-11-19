import { useDispatch } from 'react-redux';

import { actions as integrationsActions } from 'reducers/integrations';
import { actions as authActions } from 'reducers/auth';
import { actions as apitokensActions } from 'reducers/apitokens';
import { actions as machineAccessConfigsActions } from 'reducers/machineAccessConfigs';
import { actions as cloudSourcesActions } from 'reducers/cloudSources';
import type { IntegrationSource } from '../utils/integrationUtils';

const fetchIntegrationsActionMap = {
    authProviders: authActions.fetchAuthProviders.request(),
    backups: integrationsActions.fetchBackups.request(),
    imageIntegrations: integrationsActions.fetchImageIntegrations.request(),
    signatureIntegrations: integrationsActions.fetchSignatureIntegrations.request(),
    notifiers: integrationsActions.fetchNotifiers.request(),
    cloudSources: cloudSourcesActions.fetchCloudSources.request(),
};

export type UseFetchIntegrationsResponse = () => void;

const useFetchIntegrations = (source: IntegrationSource): UseFetchIntegrationsResponse => {
    const dispatch = useDispatch();

    function fetchIntegrations() {
        if (source === 'authProviders') {
            dispatch(apitokensActions.fetchAPITokens.request());
            dispatch(machineAccessConfigsActions.fetchMachineAccessConfigs.request());
        } else {
            dispatch(fetchIntegrationsActionMap[source]);
        }
    }

    return fetchIntegrations;
};

export default useFetchIntegrations;
