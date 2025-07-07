import React from 'react';
import {
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
    List,
    ListItem,
} from '@patternfly/react-core';

import ClusterHealthPanel from '../ClusterHealthPanel';
import { ClusterHealthStatus } from '../../clusterTypes';
import CollectorStatus from './CollectorStatus';

export type CollectorPanelProps = {
    healthStatus: ClusterHealthStatus;
};

function CollectorPanel({ healthStatus }: CollectorPanelProps) {
    const { collectorHealthInfo, collectorHealthStatus } = healthStatus;

    let statusMessage: string | null = null;

    if (collectorHealthStatus === 'UNINITIALIZED') {
        statusMessage = 'Uninitialized';
    } else if (collectorHealthStatus === 'UNAVAILABLE') {
        statusMessage = 'Upgrade Sensor to get Collector health information';
    }

    return (
        <ClusterHealthPanel header={<CollectorStatus healthStatus={healthStatus} />}>
            <DescriptionList>
                {statusMessage ? (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Status</DescriptionListTerm>
                        <DescriptionListDescription>{statusMessage}</DescriptionListDescription>
                    </DescriptionListGroup>
                ) : (
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
                    </>
                )}
            </DescriptionList>
        </ClusterHealthPanel>
    );
}

export default CollectorPanel;
