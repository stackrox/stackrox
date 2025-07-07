import React from 'react';
import {
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
    List,
    ListItem,
} from '@patternfly/react-core';

import AdmissionControlStatus from './AdmissionControlStatus';
import ClusterHealthPanel from '../ClusterHealthPanel';
import { ClusterHealthStatus } from '../../clusterTypes';

export type AdmissionControlPanelProps = {
    healthStatus: ClusterHealthStatus;
};

function AdmissionControlPanel({ healthStatus }: AdmissionControlPanelProps) {
    const { admissionControlHealthInfo, admissionControlHealthStatus } = healthStatus;

    let statusMessage: string | null = null;

    if (admissionControlHealthStatus === 'UNINITIALIZED') {
        statusMessage = 'Uninitialized';
    } else if (admissionControlHealthStatus === 'UNAVAILABLE') {
        statusMessage = 'Upgrade Sensor to get Admission Control health information';
    }

    return (
        <ClusterHealthPanel header={<AdmissionControlStatus healthStatus={healthStatus} />}>
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
                                {admissionControlHealthInfo?.totalReadyPods ?? 'n/a'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Pods expected</DescriptionListTerm>
                            <DescriptionListDescription>
                                {admissionControlHealthInfo?.totalDesiredPods ?? 'n/a'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        {admissionControlHealthInfo?.statusErrors &&
                            admissionControlHealthInfo.statusErrors.length > 0 && (
                                <DescriptionListGroup>
                                    <DescriptionListTerm>Errors</DescriptionListTerm>
                                    <DescriptionListDescription>
                                        <List>
                                            {admissionControlHealthInfo.statusErrors.map(
                                                (error) => (
                                                    <ListItem key={error}>{error}</ListItem>
                                                )
                                            )}
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

export default AdmissionControlPanel;
