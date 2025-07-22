import React, { ReactElement } from 'react';

import { ClusterHealthStatus } from 'types/cluster.proto';

import {
    delayedAdmissionControlStatusStyle,
    healthStatusStyles,
    isDelayedSensorHealthStatus,
} from '../../cluster.helpers';
import HealthLabelWithDelayed from '../HealthLabelWithDelayed';
import HealthStatus from '../HealthStatus';
import HealthStatusNotApplicable from '../HealthStatusNotApplicable';

type AdmissionControlStatusProps = {
    healthStatus: ClusterHealthStatus;
};

function AdmissionControlStatus({ healthStatus }: AdmissionControlStatusProps): ReactElement {
    if (!healthStatus?.admissionControlHealthStatus) {
        return <HealthStatusNotApplicable testId="admissionControlStatus" isList />;
    }

    const { admissionControlHealthStatus, sensorHealthStatus, lastContact } = healthStatus;
    const isDelayed = !!(lastContact && isDelayedSensorHealthStatus(sensorHealthStatus));
    const { Icon, fgColor } = isDelayed
        ? delayedAdmissionControlStatusStyle
        : healthStatusStyles[admissionControlHealthStatus];
    const icon = <Icon className="inline h-4 w-4" />;

    const healthLabelElement = (
        <HealthLabelWithDelayed
            isDelayed={isDelayed}
            delayedText=""
            clusterHealthItem="admissionControl"
            clusterHealthItemStatus={admissionControlHealthStatus}
            isList
        />
    );

    return (
        <HealthStatus icon={icon} iconColor={fgColor} isList>
            {healthLabelElement}
        </HealthStatus>
    );
}

export default AdmissionControlStatus;
