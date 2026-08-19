import { useState } from 'react';

import useRestMutation from 'hooks/useRestMutation';
import { fetchDeploymentRiskSummary } from 'services/DeploymentsService';
import type { DeploymentRiskSummary } from 'services/DeploymentsService';

export type UseAiRiskSummaryReturn = {
    /** Whether the AI risk briefing section is currently open */
    isOpen: boolean;
    /** The fetched risk summary, if any */
    summary: DeploymentRiskSummary | undefined;
    /** Whether the summary request is in flight */
    isLoading: boolean;
    /** The error, if the summary request failed */
    error: unknown;
    /** Open the briefing and (re)fetch the summary for the deployment */
    investigate: () => void;
    /** Close the briefing */
    close: () => void;
};

/**
 * Drives the on-demand "Investigate with Lightspeed" AI risk briefing for a deployment.
 * The summary is fetched only when the user opts in via `investigate`, not on mount.
 */
export default function useAiRiskSummary(deploymentId: string): UseAiRiskSummaryReturn {
    const [isOpen, setIsOpen] = useState(false);
    const { data, isLoading, error, mutate } = useRestMutation(fetchDeploymentRiskSummary);

    function investigate() {
        setIsOpen(true);
        mutate(deploymentId);
    }

    function close() {
        setIsOpen(false);
    }

    return { isOpen, summary: data, isLoading, error, investigate, close };
}
