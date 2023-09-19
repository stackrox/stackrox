import { useCallback } from 'react';

import useRestMutation, { UseRestMutationReturn } from 'hooks/useRestMutation';
import useRestQuery from 'hooks/useRestQuery';
import {
    VulnerabilitiesDeferralConfig,
    fetchVulnerabilitiesDeferralConfig,
    updateVulnerabilitiesDeferralConfig,
} from 'services/DeferralConfigService';

export type UseVulnerabilitiesDeferralConfigReturn = {
    config: VulnerabilitiesDeferralConfig | undefined;
    isConfigLoading: boolean;
    configLoadError: unknown;
    isUpdateInProgress: boolean;
    updateConfig: UseRestMutationReturn<
        Partial<VulnerabilitiesDeferralConfig>,
        Partial<VulnerabilitiesDeferralConfig>
    >['mutate'];
};

export function useVulnerabilitiesDeferralConfig(): UseVulnerabilitiesDeferralConfigReturn {
    const fetchConfigFn = useCallback(fetchVulnerabilitiesDeferralConfig, []);
    const getConfigRequest = useRestQuery(fetchConfigFn);
    const updateConfigMutation = useRestMutation(updateVulnerabilitiesDeferralConfig, {
        onSuccess: getConfigRequest.refetch,
    });

    return {
        config: getConfigRequest.data ?? undefined,
        isConfigLoading: getConfigRequest.loading,
        configLoadError: getConfigRequest.error,
        isUpdateInProgress: updateConfigMutation.isLoading,
        updateConfig: updateConfigMutation.mutate,
    };
}
