import type { ReactElement } from 'react';
import { Alert, Card, CardBody } from '@patternfly/react-core';
import { Table, Tbody, Th, Tr } from '@patternfly/react-table';

import type { Cluster } from 'types/cluster.proto';

import { getSensorCompatibilityCounts } from './ClustersHealth.utils';
import ClustersHealthCardHeader from './ClustersHealthCardHeader';
import {
    TdDegraded,
    TdHealthy,
    TdTotal,
    TdUnhealthy,
    TheadClustersHealth,
} from './ClustersHealthTable';

const dataLabelHealthy = 'Compatible';
const dataLabelUnhealthy = 'Incompatible';
const dataLabelDegraded = 'Unknown';

export type SensorCompatibilityCardProps = {
    clusters: Cluster[];
    compatibleVersions: string[];
    isFetchingInitialRequest: boolean;
    errorMessageFetching: string;
};

function SensorCompatibilityCard({
    clusters,
    compatibleVersions,
    isFetchingInitialRequest,
    errorMessageFetching,
}: SensorCompatibilityCardProps): ReactElement {
    const counts =
        !isFetchingInitialRequest && !errorMessageFetching
            ? getSensorCompatibilityCounts(clusters, compatibleVersions)
            : null;

    return (
        <Card isCompact>
            <ClustersHealthCardHeader
                counts={counts}
                isFetchingInitialRequest={isFetchingInitialRequest}
                title="Sensor compatibility status"
            />
            {errorMessageFetching ? (
                <CardBody>
                    <Alert isInline variant="warning" title={errorMessageFetching} component="p" />
                </CardBody>
            ) : counts !== null &&
              (counts.HEALTHY === 0 || counts.UNHEALTHY !== 0 || counts.DEGRADED !== 0) ? (
                <CardBody>
                    <Table variant="compact">
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
                                <TdTotal count={clusters.length} />
                            </Tr>
                        </Tbody>
                    </Table>
                </CardBody>
            ) : null}
        </Card>
    );
}

export default SensorCompatibilityCard;
