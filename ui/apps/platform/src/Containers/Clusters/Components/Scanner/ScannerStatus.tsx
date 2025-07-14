import React from 'react';

import { ClusterHealthStatus } from 'types/cluster.proto';
import { getDistanceStrict } from 'utils/dateUtils';

import {
    delayedScannerStatusStyle,
    healthStatusStyles,
    isDelayedSensorHealthStatus,
} from '../../cluster.helpers';
import HealthLabelWithDelayed from '../HealthLabelWithDelayed';
import HealthStatus from '../HealthStatus';
import HealthStatusNotApplicable from '../HealthStatusNotApplicable';

type ScannerStatusProps = {
    healthStatus: ClusterHealthStatus;
};

const ScannerStatus = ({ healthStatus }: ScannerStatusProps) => {
    if (!healthStatus?.scannerHealthStatus) {
        return <HealthStatusNotApplicable testId="scannerStatus" isList />;
    }
    const { scannerHealthStatus, sensorHealthStatus, lastContact } = healthStatus;
    const isDelayed = !!(lastContact && isDelayedSensorHealthStatus(sensorHealthStatus));
    const delayedText = isDelayed ? `(${getDistanceStrict(lastContact, new Date())} ago)` : '';
    const { Icon, fgColor } = isDelayed
        ? delayedScannerStatusStyle
        : healthStatusStyles[scannerHealthStatus];
    const icon = <Icon className="inline h-4 w-4" />;

    const healthLabelElement = (
        <HealthLabelWithDelayed
            isDelayed={isDelayed}
            delayedText={delayedText}
            clusterHealthItem="scanner"
            clusterHealthItemStatus={scannerHealthStatus}
            isList
        />
    );

    return (
        <HealthStatus icon={icon} iconColor={fgColor} isList>
            {healthLabelElement}
        </HealthStatus>
    );
};

export default ScannerStatus;
