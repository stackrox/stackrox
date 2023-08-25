import React, { ReactElement } from 'react';
import { Alert, Card, CardBody } from '@patternfly/react-core';
import { TableComposable, Tbody, Th, Tr } from '@patternfly/react-table';

import { Cluster } from 'types/cluster.proto';

import { getClusterBecauseOfStatusCounts, getClusterStatusCounts } from './ClustersHealth.utils';
import ClustersHealthCardHeader from './ClustersHealthCardHeader';
import {
    TdDegraded,
    TdHealthy,
    TdTotal,
    TdUnavailable,
    TdUninitialized,
    TdUnhealthy,
    TheadClustersHealth,
} from './ClustersHealthTable';

export type ClusterStatusTableProps = {
    clusters: Cluster[];
    isFetchingInitialRequest: boolean;
    errorMessageFetching: string;
};

function ClusterStatusTable({
    clusters,
    isFetchingInitialRequest,
    errorMessageFetching,
}: ClusterStatusTableProps): ReactElement {
    const countsOverall =
        !isFetchingInitialRequest && !errorMessageFetching
            ? getClusterStatusCounts(clusters)
            : null;
    const countsSensor =
        !isFetchingInitialRequest && !errorMessageFetching
            ? getClusterBecauseOfStatusCounts(clusters, 'sensorHealthStatus')
            : null;
    const countsCollector =
        !isFetchingInitialRequest && !errorMessageFetching
            ? getClusterBecauseOfStatusCounts(clusters, 'collectorHealthStatus')
            : null;
    const countsAdmissionControl =
        !isFetchingInitialRequest && !errorMessageFetching
            ? getClusterBecauseOfStatusCounts(clusters, 'admissionControlHealthStatus')
            : null;

    /*
     * Render card header without body:
     * with spinner, if fetching first request
     * with check mark, if countsOverall are healthy: HEALTHY !== 0 and UNHEALTHY === 0 && DEGRADED === 0
     *
     * Render card body:
     * for request error
     * for table of countsOverall if not healthy: HEALTHY === 0 || UNHEALTHY !== 0 || DEGRADED !== 0
     */

    /* eslint-disable no-nested-ternary */
    return (
        <Card isCompact>
            <ClustersHealthCardHeader
                counts={countsOverall}
                isFetchingInitialRequest={isFetchingInitialRequest}
                title="Cluster status"
            />
            {errorMessageFetching ? (
                <CardBody>
                    <Alert isInline variant="warning" title={errorMessageFetching} />
                </CardBody>
            ) : countsOverall !== null &&
              countsSensor !== null &&
              countsCollector !== null &&
              countsAdmissionControl !== null &&
              (countsOverall.HEALTHY === 0 ||
                  countsOverall.UNHEALTHY !== 0 ||
                  countsOverall.DEGRADED !== 0) ? (
                <CardBody>
                    <TableComposable variant="compact">
                        <TheadClustersHealth />
                        <Tbody>
                            <Tr>
                                <Th scope="row">Clusters: overall status</Th>
                                <TdHealthy count={countsOverall.HEALTHY} />
                                <TdUnhealthy count={countsOverall.UNHEALTHY} />
                                <TdDegraded count={countsOverall.DEGRADED} />
                                <TdUnavailable count={countsOverall.UNAVAILABLE} />
                                <TdUninitialized count={countsOverall.UNINITIALIZED} />
                                <TdTotal count={clusters.length} />
                            </Tr>
                            <Tr>
                                <Th scope="row">Clusters: sensor status</Th>
                                <TdHealthy count={countsSensor.HEALTHY} />
                                <TdUnhealthy count={countsSensor.UNHEALTHY} />
                                <TdDegraded count={countsSensor.DEGRADED} />
                                <TdUnavailable count={countsSensor.UNAVAILABLE} />
                                <TdUninitialized count={countsSensor.UNINITIALIZED} />
                                <TdTotal count={clusters.length} />
                            </Tr>
                            <Tr>
                                <Th scope="row">Clusters: collector status</Th>
                                <TdHealthy count={countsCollector.HEALTHY} />
                                <TdUnhealthy count={countsCollector.UNHEALTHY} />
                                <TdDegraded count={countsCollector.DEGRADED} />
                                <TdUnavailable count={countsCollector.UNAVAILABLE} />
                                <TdUninitialized count={countsCollector.UNINITIALIZED} />
                                <TdTotal count={clusters.length} />
                            </Tr>
                            <Tr>
                                <Th scope="row">Clusters: admission control status</Th>
                                <TdHealthy count={countsAdmissionControl.HEALTHY} />
                                <TdUnhealthy count={countsAdmissionControl.UNHEALTHY} />
                                <TdDegraded count={countsAdmissionControl.DEGRADED} />
                                <TdUnavailable count={countsAdmissionControl.UNAVAILABLE} />
                                <TdUninitialized count={countsAdmissionControl.UNINITIALIZED} />
                                <TdTotal count={clusters.length} />
                            </Tr>
                        </Tbody>
                    </TableComposable>
                </CardBody>
            ) : null}
        </Card>
    );
    /* eslint-enable no-nested-ternary */
}

export default ClusterStatusTable;
