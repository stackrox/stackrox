import React from 'react';
import type { ReactElement } from 'react';

import type { ClusterHealthStatus } from 'types/cluster.proto';

import { healthStatusStyles, isDelayedSensorHealthStatus } from '../cluster.helpers';

import HealthLabelWithDelayed from './HealthLabelWithDelayed';
import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';

type SensorStatusProps = {
    healthStatus: ClusterHealthStatus;
};

function SensorStatus({ healthStatus }: SensorStatusProps): ReactElement {
    if (!healthStatus?.sensorHealthStatus) {
        return <HealthStatusNotApplicable testId="sensorStatus" isList />;
    }

    const { sensorHealthStatus, lastContact } = healthStatus;

    const isDelayed = !!(lastContact && isDelayedSensorHealthStatus(sensorHealthStatus));
    const { Icon, fgColor } = healthStatusStyles[sensorHealthStatus];
    const icon = <Icon className="inline h-4 w-4" />;

    const statusElement = (
        <HealthLabelWithDelayed
            isDelayed={isDelayed}
            isList
            clusterHealthItem="sensor"
            clusterHealthItemStatus={sensorHealthStatus}
            delayedText=""
        />
    );

    return (
        <HealthStatus icon={icon} iconColor={fgColor} isList>
            {statusElement}
        </HealthStatus>
    );
}

export default SensorStatus;
