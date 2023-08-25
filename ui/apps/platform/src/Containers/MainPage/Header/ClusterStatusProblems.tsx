import React, { ReactElement } from 'react';
import { gql, useQuery } from '@apollo/client';
import Raven from 'raven-js';

import ClusterStatusButton from './ClusterStatusButton';

const CLUSTER_HEALTH_COUNTER = gql`
    query healths($query: String) {
        results: clusterHealthCounter(query: $query) {
            total
            uninitialized
            healthy
            degraded
            unhealthy
        }
    }
`;

const ClusterStatusProblems = (): ReactElement => {
    const { error, data } = useQuery(CLUSTER_HEALTH_COUNTER, { pollInterval: 30000 });

    if (error) {
        Raven.captureException(error);
    }

    return (
        <ClusterStatusButton
            degraded={data?.results?.degraded}
            unhealthy={data?.results?.unhealthy}
        />
    );
};

export default ClusterStatusProblems;
