import { useState } from 'react';

import useRestMutation from 'hooks/useRestMutation';
import type { DeploymentRiskSummary } from 'services/DeploymentsService';

import {
    fetchCachedDeploymentRiskSummary,
    peekCachedDeploymentRiskSummary,
} from './riskSummaryCache';

export type UseAiRiskSummaryReturn = {
    isOpen: boolean;
    summary: DeploymentRiskSummary | undefined;
    isLoading: boolean;
    error: unknown;
    investigate: () => void;
    close: () => void;
};

export default function useAiRiskSummary(deploymentId: string): UseAiRiskSummaryReturn {
    const [isOpen, setIsOpen] = useState(false);
    const { isLoading, error, mutate } = useRestMutation(fetchCachedDeploymentRiskSummary);

    // Read the summary from the session cache if it exists
    const summary = peekCachedDeploymentRiskSummary(deploymentId);

    function investigate() {
        setIsOpen(true);
        // Already cached this session, or a request is in flight - nothing to fetch.
        if (summary || isLoading) {
            return;
        }
        mutate(deploymentId);
    }

    function close() {
        setIsOpen(false);
    }

    return { isOpen, summary, isLoading, error, investigate, close };
}
