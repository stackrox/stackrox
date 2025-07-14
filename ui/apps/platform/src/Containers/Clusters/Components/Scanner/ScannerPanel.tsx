import React from 'react';
import {
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
    List,
    ListItem,
} from '@patternfly/react-core';

import { healthStatusLabels } from 'messages/common';
import { ClusterHealthStatus } from 'types/cluster.proto';

import ClusterHealthPanel from '../ClusterHealthPanel';
import ScannerStatus from './ScannerStatus';

export type ScannerPanelProps = {
    healthStatus: ClusterHealthStatus;
};

function resolveDbHealthStatus(desiredPods?: number, readyPods?: number): string {
    if (!desiredPods) {
        return 'n/a';
    }
    return healthStatusLabels[readyPods ? 'HEALTHY' : 'UNHEALTHY'];
}

function ScannerPanel({ healthStatus }: ScannerPanelProps) {
    const { scannerHealthInfo, scannerHealthStatus } = healthStatus;

    return (
        <ClusterHealthPanel header={<ScannerStatus healthStatus={healthStatus} />}>
            <DescriptionList>
                <DescriptionListGroup>
                    <DescriptionListTerm>Status</DescriptionListTerm>
                    <DescriptionListDescription>
                        {healthStatusLabels[scannerHealthStatus]}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                {scannerHealthInfo && (
                    <>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Pods ready</DescriptionListTerm>
                            <DescriptionListDescription>
                                {scannerHealthInfo?.totalReadyAnalyzerPods ?? 'n/a'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Pods expected</DescriptionListTerm>
                            <DescriptionListDescription>
                                {scannerHealthInfo?.totalDesiredAnalyzerPods ?? 'n/a'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Database</DescriptionListTerm>
                            <DescriptionListDescription>
                                {resolveDbHealthStatus(
                                    scannerHealthInfo?.totalDesiredDbPods,
                                    scannerHealthInfo?.totalReadyDbPods
                                )}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    </>
                )}
                {scannerHealthInfo?.statusErrors && scannerHealthInfo.statusErrors.length > 0 && (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Errors</DescriptionListTerm>
                        <DescriptionListDescription>
                            <List>
                                {scannerHealthInfo.statusErrors.map((err) => (
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

export default ScannerPanel;
