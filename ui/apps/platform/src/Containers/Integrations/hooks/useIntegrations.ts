import { useCallback, useMemo } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import { fetchIntegration, isServiceIntegrationSource } from 'services/IntegrationsService';
import { fetchAPITokens } from 'services/APITokensService';
import { fetchMachineAccessConfigs } from 'services/MachineAccessService';
import { fetchCloudSources } from 'services/CloudSourceService';

import { ensureExhaustive } from 'utils/type.utils';

import type { Integration, IntegrationSource, IntegrationType } from '../utils/integrationUtils';

export type UseIntegrationsParams = {
    source: IntegrationSource;
    type: IntegrationType;
};

export type UseIntegrationsReturn = {
    integrations: Integration[];
    isLoading: boolean;
    error: Error | undefined;
    refetch: () => void;
};

/* eslint-disable @typescript-eslint/no-unsafe-return */
function extractIntegrations(
    source: IntegrationSource,
    type: IntegrationType,
    // TODO Clean up response types with generics here to avoid `any`
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    data: Record<string, any>
): Integration[] {
    const typeLowerMatches = (integration: Integration) =>
        integration.type.toLowerCase() === type.toLowerCase();

    switch (source) {
        case 'authProviders': {
            if (type === 'apitoken') {
                return data.tokens ?? [];
            }
            if (type === 'machineAccess') {
                return data.configs ?? [];
            }
            return [];
        }
        case 'notifiers':
            return (data.notifiers ?? []).filter(typeLowerMatches);
        case 'backups':
            return (data.externalBackups ?? []).filter(typeLowerMatches);
        case 'imageIntegrations':
            return (data.integrations ?? []).filter(typeLowerMatches);
        case 'signatureIntegrations':
            return data.integrations ?? [];
        case 'cloudSources': {
            const cloudSources = data.cloudSources ?? [];
            if (type === 'paladinCloud') {
                return cloudSources.filter(
                    (integration) => (integration as { type: string }).type === 'TYPE_PALADIN_CLOUD'
                );
            }
            if (type === 'ocm') {
                return cloudSources.filter(
                    (integration) => (integration as { type: string }).type === 'TYPE_OCM'
                );
            }
            return cloudSources;
        }
        case 'apiClients':
            return [];
        default:
            return ensureExhaustive(source);
    }
}
/* eslint-enable @typescript-eslint/no-unsafe-return */

const useIntegrations = ({ source, type }: UseIntegrationsParams): UseIntegrationsReturn => {
    const fetchFn = useCallback(() => {
        if (source === 'authProviders') {
            if (type === 'apitoken') {
                return fetchAPITokens();
            }
            if (type === 'machineAccess') {
                return fetchMachineAccessConfigs();
            }
        }
        if (source === 'cloudSources') {
            return fetchCloudSources();
        }
        if (source === 'apiClients') {
            return Promise.resolve<Record<string, unknown>>({});
        }
        if (isServiceIntegrationSource(source)) {
            return fetchIntegration(source);
        }
        return ensureExhaustive(source);
    }, [source, type]);

    const { data, isLoading, error, refetch } = useRestQuery(fetchFn);

    const integrations = useMemo(
        () => (data ? extractIntegrations(source, type, data) : []),
        [source, type, data]
    );

    return { integrations, isLoading, error, refetch };
};

export default useIntegrations;
