import React, { ReactElement } from 'react';

import { ClusterHealthStatus } from 'types/cluster.proto';

import { getDistanceStrict } from 'utils/dateUtils';
import {
    delayedCollectorStatusStyle,
    healthStatusStyles,
    isDelayedSensorHealthStatus,
} from '../../cluster.helpers';
import HealthLabelWithDelayed from '../HealthLabelWithDelayed';
import HealthStatus from '../HealthStatus';
import HealthStatusNotApplicable from '../HealthStatusNotApplicable';

type CollectorStatusProps = {
    healthStatus: ClusterHealthStatus;
};

function CollectorStatus({ healthStatus }: CollectorStatusProps): ReactElement {
    if (!healthStatus?.collectorHealthStatus) {
        return <HealthStatusNotApplicable testId="collectorStatus" isList />;
    }

    const { collectorHealthStatus, sensorHealthStatus, lastContact } = healthStatus;

    const isDelayed = !!(lastContact && isDelayedSensorHealthStatus(sensorHealthStatus));
    const delayedText = isDelayed ? `(${getDistanceStrict(lastContact, new Date())} ago)` : '';
    const { Icon, fgColor } = isDelayed
        ? delayedCollectorStatusStyle
        : healthStatusStyles[collectorHealthStatus];
    const icon = <Icon className="inline h-4 w-4" />;

    const statusElement = (
        <HealthLabelWithDelayed
            isDelayed={isDelayed}
            isList
            clusterHealthItem="collector"
            clusterHealthItemStatus={collectorHealthStatus}
            delayedText={delayedText}
        />
    );

    return (
        <HealthStatus icon={icon} iconColor={fgColor} isList>
            {statusElement}
        </HealthStatus>
    );
}

export default CollectorStatus;
