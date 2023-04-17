import React from 'react';
import { Tooltip } from '@patternfly/react-core';

import DetailedTooltipContent from 'Components/DetailedTooltipContent';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';
import { ClusterHealthStatus } from '../../clusterTypes';
import {
    delayedScannerStatusStyle,
    healthStatusStyles,
    isDelayedSensorHealthStatus,
} from '../../cluster.helpers';
import HealthLabelWithDelayed from '../HealthLabelWithDelayed';
import HealthStatusNotApplicable from '../HealthStatusNotApplicable';
import HealthStatus from '../HealthStatus';
import ScannerStatusTotals from './ScannerStatusTotals';
import ScannerUnavailableStatus from './ScannerUnavailableStatus';

/*
 * Scanner Status in Clusters list if `isList={true}` or Cluster side panel if `isList={false}`
 *
 * Caller is responsible for optional chaining in case healthStatus is null.
 */

type ScannerStatusProps = {
    healthStatus: ClusterHealthStatus;
    isList?: boolean;
};

const ScannerStatus = ({ healthStatus, isList = false }: ScannerStatusProps) => {
    if (!healthStatus?.scannerHealthStatus) {
        return <HealthStatusNotApplicable testId="scannerStatus" isList={isList} />;
    }
    const {
        scannerHealthStatus,
        scannerHealthInfo,
        healthInfoComplete,
        sensorHealthStatus,
        lastContact,
    } = healthStatus;
    const isDelayed = !!(lastContact && isDelayedSensorHealthStatus(sensorHealthStatus));
    const { Icon, fgColor } = isDelayed
        ? delayedScannerStatusStyle
        : healthStatusStyles[scannerHealthStatus];
    const icon = <Icon className={`${isList ? 'inline' : ''} h-4 w-4`} />;
    const currentDatetime = new Date();

    const healthLabelElement = (
        <HealthLabelWithDelayed
            isDelayed={isDelayed}
            delayedText={getDistanceStrictAsPhrase(lastContact, currentDatetime)}
            clusterHealthItem="scanner"
            clusterHealthItemStatus={scannerHealthStatus}
            isList={isList}
        />
    );

    const healthStatusElement = (
        <HealthStatus icon={icon} iconColor={fgColor} isList={isList}>
            {healthLabelElement}
        </HealthStatus>
    );

    if (scannerHealthInfo) {
        const scannerTotalsElement = <ScannerStatusTotals scannerHealthInfo={scannerHealthInfo} />;
        const infoElement = healthInfoComplete ? (
            scannerTotalsElement
        ) : (
            <div>
                {scannerTotalsElement}
                <div data-testid="scannerInfoComplete">
                    <strong>Upgrade Sensor</strong> to get complete Scanner health information
                </div>
            </div>
        );

        return isList ? (
            <Tooltip
                content={
                    <DetailedTooltipContent title="Scanner Health Information" body={infoElement} />
                }
            >
                <div className="inline">{healthStatusElement}</div>
            </Tooltip>
        ) : (
            <HealthStatus icon={icon} iconColor={fgColor}>
                <div>
                    {healthLabelElement}
                    {infoElement}
                </div>
            </HealthStatus>
        );
    }

    if (scannerHealthStatus === 'UNAVAILABLE') {
        return (
            <ScannerUnavailableStatus
                isList={isList}
                icon={icon}
                fgColor={fgColor}
                healthStatusElement={healthStatusElement}
                healthLabelElement={healthLabelElement}
            />
        );
    }

    // UNINITIALIZED
    return healthStatusElement;
};

export default ScannerStatus;
