import { useState } from 'react';

import useRestMutation from 'hooks/useRestMutation';
import type { DeploymentRiskSummary } from 'services/DeploymentsService';

import {
    fetchCachedDeploymentRiskSummary,
    peekCachedDeploymentRiskSummary,
} from './riskSummaryCache';

export type UseAiRiskSummaryReturn = {
    isPresent: boolean;
    isExpanded: boolean;
    summary: DeploymentRiskSummary | undefined;
    isLoading: boolean;
    error: unknown;
    investigate: () => void;
    setExpanded: (isExpanded: boolean) => void;
};

export default function useAiRiskSummary(deploymentId: string): UseAiRiskSummaryReturn {
    const [isExpanded, setIsExpanded] = useState(false);
    const { isLoading, error, mutate } = useRestMutation(fetchCachedDeploymentRiskSummary);

    // Read the summary from the session cache if it exists
    const summary = peekCachedDeploymentRiskSummary(deploymentId);

    // The card is present (collapsed or expanded) whenever there is a cached result to
    // surface, a request in flight, an error to report, or the user has expanded it this
    // session. A cached result on a fresh mount renders the card collapsed, so navigating
    // back to a deployment shows that recent results exist without re-spending tokens.
    const isPresent = Boolean(summary) || isLoading || Boolean(error) || isExpanded;

    function investigate() {
        setIsExpanded(true);
        // Already cached this session, or a request is in flight - nothing to fetch.
        if (summary || isLoading) {
            return;
        }
        mutate(deploymentId);
    }

    return {
        isPresent,
        isExpanded,
        summary,
        isLoading,
        error,
        investigate,
        setExpanded: setIsExpanded,
    };
}
