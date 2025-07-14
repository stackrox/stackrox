import React from 'react';
import {
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
} from '@patternfly/react-core';
import { healthStatusLabels } from 'messages/common';

import { ClusterHealthStatus } from 'types/cluster.proto';
import { getDateTime, getDistanceStrictAsPhrase } from 'utils/dateUtils';

import { isDelayedSensorHealthStatus } from '../cluster.helpers';
import ClusterHealthPanel from './ClusterHealthPanel';
import SensorStatus from './SensorStatus';

export type SensorPanelProps = {
    healthStatus: ClusterHealthStatus;
};

function SensorPanel({ healthStatus }: SensorPanelProps) {
    const { lastContact, sensorHealthStatus } = healthStatus;

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
                <DescriptionListGroup>
                    <DescriptionListTerm>Status</DescriptionListTerm>
                    <DescriptionListDescription>
                        {healthStatusLabels[sensorHealthStatus]}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                {sensorHealthStatus !== 'UNINITIALIZED' && (
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
