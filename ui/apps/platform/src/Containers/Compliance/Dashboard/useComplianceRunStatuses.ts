import { useState } from 'react';
import { DocumentNode, gql, useApolloClient, useQuery } from '@apollo/client';

export type ComplianceRunStatusResponse = {
    complianceRunStatuses: {
        runs: { state: string }[];
    };
};

function isCurrentScanIncomplete(
    runs: ComplianceRunStatusResponse['complianceRunStatuses']['runs']
) {
    return runs.some((run) => run.state !== 'FINISHED');
}

const complianceRunStatusesQuery = gql`
    query runStatuses($latest: Boolean) {
        complianceRunStatuses(latest: $latest) {
            runs {
                state
            }
        }
    }
`;

const pollInterval = 10000;
const variables = {
    latest: true,
};

export type UseComplianceRunStatusesResponse = {
    /* The latest compliance scan runs */
    runs: ComplianceRunStatusResponse['complianceRunStatuses']['runs'];
    error: unknown;
    /* Fetches the latest compliance scan runs, and restarts polling, if necessary */
    restartPolling: () => void;
    /* Whether or not an in progress scan was detected during the lifetime of this hook */
    inProgressScanDetected: boolean;
    /* Whether or not the latest scan is incomplete */
    isCurrentScanIncomplete: boolean;
};

export function useComplianceRunStatuses(
    queriesToRefetchOnPollingComplete: DocumentNode[]
): UseComplianceRunStatusesResponse {
    const [inProgressScanDetected, setInProgressScanDetected] = useState(false);

    const client = useApolloClient();

    const { startPolling, stopPolling, error, data, refetch } = useQuery<
        ComplianceRunStatusResponse,
        typeof variables
    >(complianceRunStatusesQuery, {
        variables,
        pollInterval,
        onCompleted,
        onError,
        errorPolicy: 'all',
        notifyOnNetworkStatusChange: true,
    });

    function onCompleted(data: ComplianceRunStatusResponse) {
        if (isCurrentScanIncomplete(data.complianceRunStatuses.runs)) {
            setInProgressScanDetected(true);
        } else {
            stopPolling();
        }

        return client.refetchQueries({ include: queriesToRefetchOnPollingComplete });
    }

    function onError() {
        stopPolling();
    }

    return {
        error,
        runs: data?.complianceRunStatuses.runs ?? [],
        restartPolling: () =>
            refetch().then(({ data }) => {
                if (isCurrentScanIncomplete(data.complianceRunStatuses.runs)) {
                    setInProgressScanDetected(true);
                    startPolling(pollInterval);
                }
            }),
        inProgressScanDetected,
        isCurrentScanIncomplete: isCurrentScanIncomplete(data?.complianceRunStatuses.runs ?? []),
    };
}
