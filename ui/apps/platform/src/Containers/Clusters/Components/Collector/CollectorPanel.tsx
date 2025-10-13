import React, { useMemo } from 'react';
import {
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
    List,
    ListItem,
} from '@patternfly/react-core';

import type { ClusterHealthStatus } from 'types/cluster.proto';
import { buildStatusMessage } from '../../cluster.helpers';

import ClusterHealthPanel from '../ClusterHealthPanel';
import CollectorStatus from './CollectorStatus';

export type CollectorPanelProps = {
    healthStatus: ClusterHealthStatus;
};

function CollectorPanel({ healthStatus }: CollectorPanelProps) {
    const { collectorHealthInfo, collectorHealthStatus, sensorHealthStatus, lastContact } =
        healthStatus;

    const statusMessage = useMemo(
        () => buildStatusMessage(collectorHealthStatus, lastContact, sensorHealthStatus),
        [collectorHealthStatus, sensorHealthStatus, lastContact]
    );

    return (
        <ClusterHealthPanel header={<CollectorStatus healthStatus={healthStatus} />}>
            <DescriptionList>
                <DescriptionListGroup>
                    <DescriptionListTerm>Status</DescriptionListTerm>
                    <DescriptionListDescription>{statusMessage}</DescriptionListDescription>
                </DescriptionListGroup>
                {collectorHealthInfo && (
                    <>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Pods ready</DescriptionListTerm>
                            <DescriptionListDescription>
                                {collectorHealthInfo?.totalReadyPods ?? 'n/a'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Pods expected</DescriptionListTerm>
                            <DescriptionListDescription>
                                {collectorHealthInfo?.totalDesiredPods ?? 'n/a'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Registered nodes in cluster</DescriptionListTerm>
                            <DescriptionListDescription>
                                {collectorHealthInfo?.totalRegisteredNodes ?? 'n/a'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Version</DescriptionListTerm>
                            <DescriptionListDescription>
                                {collectorHealthInfo?.version ?? 'n/a'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    </>
                )}
                {collectorHealthStatus === 'UNAVAILABLE' && (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Notes</DescriptionListTerm>
                        <DescriptionListDescription>
                            Upgrade Sensor to get Collector health information
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                )}
                {collectorHealthInfo?.statusErrors &&
                    collectorHealthInfo.statusErrors.length > 0 && (
                        <DescriptionListGroup>
                            <DescriptionListTerm>Errors</DescriptionListTerm>
                            <DescriptionListDescription>
                                <List>
                                    {collectorHealthInfo.statusErrors.map((err) => (
                                        <ListItem key={err}>{err}</ListItem>
                                    ))}
                                </List>
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    )}
            </DescriptionList>
        </ClusterHealthPanel>
    );
}

export default CollectorPanel;
