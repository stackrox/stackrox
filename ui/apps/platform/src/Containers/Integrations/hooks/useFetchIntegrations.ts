import { useDispatch } from 'react-redux';

import { actions as integrationsActions } from 'reducers/integrations';
import { actions as authActions } from 'reducers/auth';
import { actions as apitokensActions } from 'reducers/apitokens';
import { actions as clusterActions } from 'reducers/clusters';
import { IntegrationSource } from '../utils/integrationUtils';

const fetchIntegrationsActionMap = {
    authPlugins: integrationsActions.fetchAuthPlugins.request(),
    authProviders: authActions.fetchAuthProviders.request(),
    backups: integrationsActions.fetchBackups.request(),
    imageIntegrations: integrationsActions.fetchImageIntegrations.request(),
    notifiers: integrationsActions.fetchNotifiers.request(),
    clusters: clusterActions.fetchClusters.request(),
    apitoken: apitokensActions.fetchAPITokens.request(),
};

export type UseFetchIntegrationsResponse = () => void;

const useFetchIntegrations = (source: IntegrationSource): UseFetchIntegrationsResponse => {
    const dispatch = useDispatch();

    function fetchIntegrations() {
        dispatch(fetchIntegrationsActionMap[source]);
    }

    return fetchIntegrations;
};

export default useFetchIntegrations;
