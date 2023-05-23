import React, { ReactElement } from 'react';
import { Alert, Card, CardBody } from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Tr } from '@patternfly/react-table';

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
    requestErrorMessage: string;
};

function ClusterStatusTable({
    clusters,
    isFetchingInitialRequest,
    requestErrorMessage,
}: ClusterStatusTableProps): ReactElement {
    const counts =
        !isFetchingInitialRequest && !requestErrorMessage ? getClusterStatusCounts(clusters) : null;
    const countsSensor =
        !isFetchingInitialRequest && !requestErrorMessage
            ? getClusterBecauseOfStatusCounts(clusters, 'sensorHealthStatus')
            : null;
    const countsCollector =
        !isFetchingInitialRequest && !requestErrorMessage
            ? getClusterBecauseOfStatusCounts(clusters, 'collectorHealthStatus')
            : null;
    const countsAdmissionControl =
        !isFetchingInitialRequest && !requestErrorMessage
            ? getClusterBecauseOfStatusCounts(clusters, 'admissionControlHealthStatus')
            : null;

    /*
     * Render card header without body:
     * with spinner, if fetching first request
     * with check mark, if counts are healthy: HEALTHY !== 0 and UNHEALTHY === 0 && DEGRADED === 0
     *
     * Render card body:
     * for request error
     * for table of counts if not healthy: HEALTHY === 0 || UNHEALTHY !== 0 || DEGRADED !== 0
     */

    /* eslint-disable no-nested-ternary */
    return (
        <Card isCompact>
            <ClustersHealthCardHeader
                counts={counts}
                isFetchingInitialRequest={isFetchingInitialRequest}
                title="Cluster status"
            />
            {requestErrorMessage ? (
                <CardBody>
                    <Alert isInline variant="warning" title={requestErrorMessage} />
                </CardBody>
            ) : counts !== null &&
              countsSensor !== null &&
              countsCollector !== null &&
              countsAdmissionControl !== null &&
              (counts.HEALTHY === 0 || counts.UNHEALTHY !== 0 || counts.DEGRADED !== 0) ? (
                <CardBody>
                    <TableComposable variant="compact">
                        <TheadClustersHealth />
                        <Tbody>
                            <Tr>
                                <Td>Clusters because any of the following</Td>
                                <TdHealthy counts={counts} />
                                <TdUnhealthy counts={counts} />
                                <TdDegraded counts={counts} />
                                <TdUnavailable counts={counts} />
                                <TdUninitialized counts={counts} />
                                <TdTotal clusters={clusters} />
                            </Tr>
                            <Tr>
                                <Td>Clusters because sensor status</Td>
                                <TdHealthy counts={countsSensor} />
                                <TdUnhealthy counts={countsSensor} />
                                <TdDegraded counts={countsSensor} />
                                <TdUnavailable counts={countsSensor} />
                                <TdUninitialized counts={countsSensor} />
                                <TdTotal clusters={clusters} />
                            </Tr>
                            <Tr>
                                <Td>Clusters because collector status</Td>
                                <TdHealthy counts={countsCollector} />
                                <TdUnhealthy counts={countsCollector} />
                                <TdDegraded counts={countsCollector} />
                                <TdUnavailable counts={countsCollector} />
                                <TdUninitialized counts={countsCollector} />
                                <TdTotal clusters={clusters} />
                            </Tr>
                            <Tr>
                                <Td>Clusters because admission control status</Td>
                                <TdHealthy counts={countsAdmissionControl} />
                                <TdUnhealthy counts={countsAdmissionControl} />
                                <TdDegraded counts={countsAdmissionControl} />
                                <TdUnavailable counts={countsAdmissionControl} />
                                <TdUninitialized counts={countsAdmissionControl} />
                                <TdTotal clusters={clusters} />
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
