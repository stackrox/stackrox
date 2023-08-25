import React, { ReactElement } from 'react';
import { Alert, Card, CardBody } from '@patternfly/react-core';
import { TableComposable, Tbody, Th, Tr } from '@patternfly/react-table';

import { Cluster } from 'types/cluster.proto';

import { getSensorUpgradeCounts } from './ClustersHealth.utils';
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

const dataLabelHealthy = 'Up to date';
const dataLabelUnhealthy = 'Failed';
const dataLabelDegraded = 'Out of date';

export type SensorUpgradeCardProps = {
    clusters: Cluster[];
    isFetchingInitialRequest: boolean;
    errorMessageFetching: string;
};

function SensorUpgradeCard({
    clusters,
    isFetchingInitialRequest,
    errorMessageFetching,
}: SensorUpgradeCardProps): ReactElement {
    const counts =
        !isFetchingInitialRequest && !errorMessageFetching
            ? getSensorUpgradeCounts(clusters)
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
                title="Sensor upgrade"
            />
            {errorMessageFetching ? (
                <CardBody>
                    <Alert isInline variant="warning" title={errorMessageFetching} />
                </CardBody>
            ) : counts !== null &&
              (counts.HEALTHY === 0 || counts.UNHEALTHY !== 0 || counts.DEGRADED !== 0) ? (
                <CardBody>
                    <TableComposable variant="compact">
                        <TheadClustersHealth
                            dataLabelHealthy={dataLabelHealthy}
                            dataLabelUnhealthy={dataLabelUnhealthy}
                            dataLabelDegraded={dataLabelDegraded}
                        />
                        <Tbody>
                            <Tr>
                                <Th scope="row">Clusters</Th>
                                <TdHealthy count={counts.HEALTHY} dataLabel={dataLabelHealthy} />
                                <TdUnhealthy
                                    count={counts.UNHEALTHY}
                                    dataLabel={dataLabelUnhealthy}
                                />
                                <TdDegraded count={counts.DEGRADED} dataLabel={dataLabelDegraded} />
                                <TdUnavailable count={counts.UNAVAILABLE} />
                                <TdUninitialized count={counts.UNINITIALIZED} />
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

export default SensorUpgradeCard;
