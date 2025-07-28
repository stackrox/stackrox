import React from 'react';
import {
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
    List,
    ListItem,
} from '@patternfly/react-core';

import { ClusterHealthStatus } from 'types/cluster.proto';
import { buildStatusMessage } from 'Containers/Clusters/cluster.helpers';

import AdmissionControlStatus from './AdmissionControlStatus';
import ClusterHealthPanel from '../ClusterHealthPanel';

export type AdmissionControlPanelProps = {
    healthStatus: ClusterHealthStatus;
};

function AdmissionControlPanel({ healthStatus }: AdmissionControlPanelProps) {
    const {
        admissionControlHealthInfo,
        admissionControlHealthStatus,
        sensorHealthStatus,
        lastContact,
    } = healthStatus;

    const statusMessage = buildStatusMessage(
        admissionControlHealthStatus,
        lastContact,
        sensorHealthStatus
    );

    return (
        <ClusterHealthPanel header={<AdmissionControlStatus healthStatus={healthStatus} />}>
            <DescriptionList>
                <DescriptionListGroup>
                    <DescriptionListTerm>Status</DescriptionListTerm>
                    <DescriptionListDescription>{statusMessage}</DescriptionListDescription>
                </DescriptionListGroup>
                {admissionControlHealthInfo && (
                    <>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Pods ready</DescriptionListTerm>
                            <DescriptionListDescription>
                                {admissionControlHealthInfo?.totalReadyPods ?? 'n/a'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Pods expected</DescriptionListTerm>
                            <DescriptionListDescription>
                                {admissionControlHealthInfo?.totalDesiredPods ?? 'n/a'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    </>
                )}
                {admissionControlHealthStatus === 'UNAVAILABLE' && (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Notes</DescriptionListTerm>
                        <DescriptionListDescription>
                            Upgrade Sensor to get Admission Control health information
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                )}
                {admissionControlHealthInfo?.statusErrors &&
                    admissionControlHealthInfo.statusErrors.length > 0 && (
                        <DescriptionListGroup>
                            <DescriptionListTerm>Errors</DescriptionListTerm>
                            <DescriptionListDescription>
                                <List>
                                    {admissionControlHealthInfo.statusErrors.map((error) => (
                                        <ListItem key={error}>{error}</ListItem>
                                    ))}
                                </List>
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    )}
            </DescriptionList>
        </ClusterHealthPanel>
    );
}

export default AdmissionControlPanel;
