import { useState } from 'react';
import { gql, useApolloClient, useQuery } from '@apollo/client';

import { resourceTypes } from 'constants/entityTypes';
import {
    AGGREGATED_RESULTS_ACROSS_ENTITY,
    AGGREGATED_RESULTS_STANDARDS_BY_ENTITY,
} from 'queries/controls';

export type ComplianceRunStatusResponse = {
    complianceRunStatuses: {
        runs: { state: string }[];
    };
};

export function isCurrentScanIncomplete(
    runs: ComplianceRunStatusResponse['complianceRunStatuses']['runs']
) {
    return runs.some((run) => run.state !== 'FINISHED');
}

const queriesToRefetchOnPollingComplete = [
    AGGREGATED_RESULTS_STANDARDS_BY_ENTITY(resourceTypes.CLUSTER),
    AGGREGATED_RESULTS_ACROSS_ENTITY(resourceTypes.CLUSTER),
    AGGREGATED_RESULTS_ACROSS_ENTITY(resourceTypes.NAMESPACE),
    AGGREGATED_RESULTS_ACROSS_ENTITY(resourceTypes.NODE),
];

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
};

export function useComplianceRunStatuses(): UseComplianceRunStatusesResponse {
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
        fetchPolicy: 'no-cache',
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
    };
}
