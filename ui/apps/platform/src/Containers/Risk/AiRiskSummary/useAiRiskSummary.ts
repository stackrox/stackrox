import { useState } from 'react';

import useAnalytics, { RISK_SUMMARY_GENERATED } from 'hooks/useAnalytics';
import useRestMutation from 'hooks/useRestMutation';
import { fetchDeploymentRiskSummary } from 'services/DeploymentsService';
import type { DeploymentRiskSummary } from 'services/DeploymentsService';

export type UseAiRiskSummaryReturn = {
    isExpanded: boolean;
    summary: DeploymentRiskSummary | undefined;
    isLoading: boolean;
    error: unknown;
    investigate: () => void;
    setExpanded: (isExpanded: boolean) => void;
};

export default function useAiRiskSummary(deploymentId: string): UseAiRiskSummaryReturn {
    const [isExpanded, setIsExpanded] = useState(false);
    const { analyticsTrack } = useAnalytics();
    const {
        data: summary,
        isLoading,
        error,
        mutate,
    } = useRestMutation(fetchDeploymentRiskSummary, {
        onSuccess: () => analyticsTrack(RISK_SUMMARY_GENERATED),
    });

    function investigate() {
        if (isLoading) {
            return;
        }
        setIsExpanded(true);
        mutate(deploymentId);
    }

    return {
        isExpanded,
        summary,
        isLoading,
        error,
        investigate,
        setExpanded: setIsExpanded,
    };
}
