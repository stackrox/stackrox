import React from 'react';
import {
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
} from '@patternfly/react-core';

import type { ClusterHealthStatus } from 'types/cluster.proto';
import { getDateTime } from 'utils/dateUtils';

import { buildStatusMessage } from '../cluster.helpers';
import ClusterHealthPanel from './ClusterHealthPanel';
import SensorStatus from './SensorStatus';

export type SensorPanelProps = {
    healthStatus: ClusterHealthStatus;
};

function SensorPanel({ healthStatus }: SensorPanelProps) {
    const { lastContact, sensorHealthStatus } = healthStatus;

    const statusMessage = buildStatusMessage(
        sensorHealthStatus,
        lastContact,
        sensorHealthStatus,
        (delayedText) => `for ${delayedText}`
    );

    return (
        <ClusterHealthPanel header={<SensorStatus healthStatus={healthStatus} />}>
            <DescriptionList>
                <DescriptionListGroup>
                    <DescriptionListTerm>Status</DescriptionListTerm>
                    <DescriptionListDescription>{statusMessage}</DescriptionListDescription>
                </DescriptionListGroup>
                {sensorHealthStatus !== 'UNINITIALIZED' && (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Last contact</DescriptionListTerm>
                        <DescriptionListDescription>
                            {getDateTime(lastContact)}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                )}
            </DescriptionList>
        </ClusterHealthPanel>
    );
}

export default SensorPanel;
