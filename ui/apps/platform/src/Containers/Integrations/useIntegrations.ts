/* eslint-disable @typescript-eslint/no-unsafe-return */
import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import { Integration, IntegrationSource, IntegrationType } from './utils/integrationUtils';

const selectIntegrations = createStructuredSelector({
    authPlugins: selectors.getAuthPlugins,
    authProviders: selectors.getAuthProviders,
    apiTokens: selectors.getAPITokens,
    clusterInitBundles: selectors.getClusterInitBundles,
    notifiers: selectors.getNotifiers,
    imageIntegrations: selectors.getImageIntegrations,
    backups: selectors.getBackups,
    featureFlags: selectors.getFeatureFlags,
});

export type UseIntegration = {
    selectedSource: IntegrationSource;
    selectedType: IntegrationType;
};

const useIntegrations = ({ selectedSource, selectedType }: UseIntegration): Integration[] => {
    const {
        authPlugins,
        apiTokens,
        clusterInitBundles,
        authProviders,
        notifiers,
        backups,
        imageIntegrations,
    } = useSelector(selectIntegrations);

    function findIntegrations(source: IntegrationSource, type: IntegrationType) {
        const typeLowerMatches = (integration: Integration) =>
            integration.type.toLowerCase() === type.toLowerCase();

        switch (source) {
            case 'authPlugins': {
                return authPlugins;
            }
            case 'authProviders': {
                if (type === 'apitoken') {
                    return apiTokens;
                }
                if (type === 'clusterInitBundle') {
                    return clusterInitBundles;
                }
                return authProviders.filter(typeLowerMatches);
            }
            case 'notifiers': {
                return notifiers.filter(typeLowerMatches);
            }
            case 'backups': {
                return backups.filter(typeLowerMatches);
            }
            case 'imageIntegrations': {
                return imageIntegrations.filter(typeLowerMatches);
            }
            default: {
                // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
                throw new Error(`Unknown source ${source}`);
            }
        }
    }

    return findIntegrations(selectedSource, selectedType);
};

export default useIntegrations;
