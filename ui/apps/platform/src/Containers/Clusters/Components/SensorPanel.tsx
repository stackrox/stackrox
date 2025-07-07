import React from 'react';
import {
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
} from '@patternfly/react-core';

import { getDateTime, getDistanceStrictAsPhrase } from 'utils/dateUtils';

import { isDelayedSensorHealthStatus } from '../cluster.helpers';
import ClusterHealthPanel from './ClusterHealthPanel';
import { ClusterHealthStatus } from '../clusterTypes';
import SensorStatus from './SensorStatus';

export type SensorPanelProps = {
    healthStatus: ClusterHealthStatus;
};

function SensorPanel({ healthStatus }: SensorPanelProps) {
    const { lastContact, sensorHealthStatus } = healthStatus;

    let statusMessage: string | null = null;

    if (sensorHealthStatus === 'UNINITIALIZED') {
        statusMessage = 'Uninitialized';
    }

    let lastContactText = 'n/a';

    if (lastContact) {
        const formatted = getDateTime(lastContact);
        if (isDelayedSensorHealthStatus(sensorHealthStatus)) {
            const distance = getDistanceStrictAsPhrase(lastContact, new Date());
            lastContactText = `${formatted} (${distance})`;
        } else {
            lastContactText = formatted;
        }
    }

    return (
        <ClusterHealthPanel header={<SensorStatus healthStatus={healthStatus} />}>
            <DescriptionList>
                {statusMessage ? (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Status</DescriptionListTerm>
                        <DescriptionListDescription>{statusMessage}</DescriptionListDescription>
                    </DescriptionListGroup>
                ) : (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Last contact</DescriptionListTerm>
                        <DescriptionListDescription>{lastContactText}</DescriptionListDescription>
                    </DescriptionListGroup>
                )}
            </DescriptionList>
        </ClusterHealthPanel>
    );
}

export default SensorPanel;
