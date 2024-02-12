/* eslint-disable @typescript-eslint/no-unsafe-return */
import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import { Integration, IntegrationSource, IntegrationType } from '../utils/integrationUtils';

const selectIntegrations = createStructuredSelector({
    apiTokens: selectors.getAPITokens,
    clusterInitBundles: selectors.getClusterInitBundles,
    machineAccessConfigs: selectors.getMachineAccessConfigs,
    notifiers: selectors.getNotifiers,
    imageIntegrations: selectors.getImageIntegrations,
    backups: selectors.getBackups,
    signatureIntegrations: selectors.getSignatureIntegrations,
    cloudSources: selectors.getCloudSources,
});

export type UseIntegrations = {
    source: IntegrationSource;
    type: IntegrationType;
};

export type UseIntegrationsResponse = Integration[];

const useIntegrations = ({ source, type }: UseIntegrations): UseIntegrationsResponse => {
    const {
        apiTokens,
        clusterInitBundles,
        machineAccessConfigs,
        notifiers,
        backups,
        imageIntegrations,
        signatureIntegrations,
        cloudSources,
    } = useSelector(selectIntegrations);

    function findIntegrations() {
        const typeLowerMatches = (integration: Integration) =>
            integration.type.toLowerCase() === type.toLowerCase();

        switch (source) {
            case 'authProviders': {
                // Integrations Authentication Tokens differ from Access Control Auth providers.
                if (type === 'apitoken') {
                    return apiTokens;
                }
                if (type === 'clusterInitBundle') {
                    return clusterInitBundles;
                }
                if (type === 'machineAccess') {
                    return machineAccessConfigs;
                }
                return [];
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
            case 'signatureIntegrations': {
                return signatureIntegrations;
            }
            case 'cloudSources': {
                return cloudSources;
            }
            default: {
                // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
                throw new Error(`Unknown source ${source}`);
            }
        }
    }

    const integrations = findIntegrations();

    return integrations;
};

export default useIntegrations;
