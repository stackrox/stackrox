import React, { ReactElement } from 'react';

import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';

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
        return <HealthStatusNotApplicable testId="sensorStatus" />;
    }

    const { sensorHealthStatus, lastContact } = healthStatus;
    const { Icon, fgColor } = healthStatusStyles[sensorHealthStatus];
    const currentDatetime = new Date();

    const isDelayed = !!(lastContact && isDelayedSensorHealthStatus(sensorHealthStatus));
    const delayedText = `for ${getDistanceStrict(lastContact, currentDatetime)}`;
    const icon = <Icon className="h-4 w-4" />;
    const sensorStatus = (
        <HealthStatus icon={icon} iconColor={fgColor}>
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
            <Tooltip
                content={
                    <TooltipOverlay>{`Last contact: ${getDateTime(lastContact)}`}</TooltipOverlay>
                }
            >
                <div>{sensorStatus}</div>
            </Tooltip>
        );
    }

    return sensorStatus;
}

export default SensorStatus;
