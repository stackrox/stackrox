import React, { ReactElement } from 'react';
import { Alert, Card, CardBody } from '@patternfly/react-core';
import { Table, Tbody, Th, Tr } from '@patternfly/react-table';

import { Cluster } from 'types/cluster.proto';

import { getCertificateExpirationCounts } from './ClustersHealth.utils';
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

const dataLabelHealthy = '\u2265 30 days'; // greater than or equal to
const dataLabelUnhealthy = '< 7 days';
const dataLabelDegraded = '< 30 days';

export type CredentialExpirationCardProps = {
    clusters: Cluster[];
    currentDatetime: Date | null;
    isFetchingInitialRequest: boolean;
    errorMessageFetching: string;
};

function CredentialExpirationCard({
    clusters,
    currentDatetime,
    isFetchingInitialRequest,
    errorMessageFetching,
}: CredentialExpirationCardProps): ReactElement {
    const counts = currentDatetime
        ? getCertificateExpirationCounts(clusters, currentDatetime)
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
                title="Credential expiration"
            />
            {errorMessageFetching ? (
                <CardBody>
                    <Alert isInline variant="warning" title={errorMessageFetching} />
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
                                <TdUnavailable count={counts.UNAVAILABLE} />
                                <TdUninitialized count={counts.UNINITIALIZED} />
                                <TdTotal count={clusters.length} />
                            </Tr>
                        </Tbody>
                    </Table>
                </CardBody>
            ) : null}
        </Card>
    );
    /* eslint-enable no-nested-ternary */
}

export default CredentialExpirationCard;
