import React, { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';

import { getDateTime, getDistanceStrict } from 'utils/dateUtils';
import HealthStatus from './HealthStatus';
import { healthStatusStyles, isDelayedSensorHealthStatus } from '../cluster.helpers';
import { ClusterHealthStatus } from '../clusterTypes';
import HealthLabelWithDelayed from './HealthLabelWithDelayed';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';

/*
 * Sensor Status in Clusters list or Cluster side panel
 *
 * Caller is responsible for optional chaining in case healthStatus is null.
 */

type SensorStatusProps = {
    healthStatus: ClusterHealthStatus;
    isList?: boolean;
};

function SensorStatus({ healthStatus, isList = false }: SensorStatusProps): ReactElement {
    if (!healthStatus?.sensorHealthStatus) {
        return <HealthStatusNotApplicable testId="sensorStatus" isList={isList} />;
    }

    const { sensorHealthStatus, lastContact } = healthStatus;
    const { Icon, fgColor } = healthStatusStyles[sensorHealthStatus];
    const currentDatetime = new Date();

    const isDelayed = !!(lastContact && isDelayedSensorHealthStatus(sensorHealthStatus));
    const delayedText = `for ${getDistanceStrict(lastContact, currentDatetime, {
        partialMethod: 'floor',
    })}`;
    const icon = <Icon className={`${isList ? 'inline' : ''} h-4 w-4`} />;
    const sensorStatus = (
        <HealthStatus icon={icon} iconColor={fgColor} isList={isList}>
            <HealthLabelWithDelayed
                clusterHealthItem="sensor"
                clusterHealthItemStatus={sensorHealthStatus}
                isList={isList}
                isDelayed={isDelayed}
                delayedText={delayedText}
            />
        </HealthStatus>
    );

    if (lastContact) {
        // Tooltip has absolute time (in ISO 8601 format) to find info from logs.
        return (
            <Tooltip content={`Last contact: ${getDateTime(lastContact)}`}>
                <div className={`${isList ? 'inline' : ''}`}>{sensorStatus}</div>
            </Tooltip>
        );
    }

    return sensorStatus;
}

export default SensorStatus;
