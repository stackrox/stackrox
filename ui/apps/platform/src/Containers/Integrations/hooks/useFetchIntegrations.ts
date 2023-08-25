import { useDispatch } from 'react-redux';

import { actions as integrationsActions } from 'reducers/integrations';
import { actions as authActions } from 'reducers/auth';
import { actions as apitokensActions } from 'reducers/apitokens';
import { actions as clusterInitBundleActions } from 'reducers/clusterInitBundles';
import { IntegrationSource } from '../utils/integrationUtils';

const fetchIntegrationsActionMap = {
    authProviders: authActions.fetchAuthProviders.request(),
    backups: integrationsActions.fetchBackups.request(),
    imageIntegrations: integrationsActions.fetchImageIntegrations.request(),
    signatureIntegrations: integrationsActions.fetchSignatureIntegrations.request(),
    notifiers: integrationsActions.fetchNotifiers.request(),
};

export type UseFetchIntegrationsResponse = () => void;

const useFetchIntegrations = (source: IntegrationSource): UseFetchIntegrationsResponse => {
    const dispatch = useDispatch();

    function fetchIntegrations() {
        if (source === 'authProviders') {
            dispatch(clusterInitBundleActions.fetchClusterInitBundles.request());
            dispatch(apitokensActions.fetchAPITokens.request());
        } else {
            dispatch(fetchIntegrationsActionMap[source]);
        }
    }

    return fetchIntegrations;
};

export default useFetchIntegrations;
