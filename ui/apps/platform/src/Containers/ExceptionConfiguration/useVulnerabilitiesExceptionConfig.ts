import useRestMutation from 'hooks/useRestMutation';
import type { UseRestMutationReturn } from 'hooks/useRestMutation';
import useRestQuery from 'hooks/useRestQuery';
import {
    fetchVulnerabilitiesExceptionConfig,
    updateVulnerabilitiesExceptionConfig,
} from 'services/ExceptionConfigService';
import type { VulnerabilitiesExceptionConfig } from 'services/ExceptionConfigService';

export type UseVulnerabilitiesExceptionConfigReturn = {
    config: VulnerabilitiesExceptionConfig | undefined;
    isConfigLoading: boolean;
    configLoadError: unknown;
    isUpdateInProgress: boolean;
    updateConfig: UseRestMutationReturn<
        Partial<VulnerabilitiesExceptionConfig>,
        Partial<VulnerabilitiesExceptionConfig>
    >['mutate'];
};

export function useVulnerabilitiesExceptionConfig(): UseVulnerabilitiesExceptionConfigReturn {
    const getConfigRequest = useRestQuery(fetchVulnerabilitiesExceptionConfig);
    const updateConfigMutation = useRestMutation(updateVulnerabilitiesExceptionConfig, {
        onSuccess: getConfigRequest.refetch,
    });

    return {
        config: getConfigRequest.data ?? undefined,
        isConfigLoading: getConfigRequest.isLoading,
        configLoadError: getConfigRequest.error,
        isUpdateInProgress: updateConfigMutation.isLoading,
        updateConfig: updateConfigMutation.mutate,
    };
}
