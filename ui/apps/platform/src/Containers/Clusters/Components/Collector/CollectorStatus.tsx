import React, { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';

import DetailedTooltipContent from 'Components/DetailedTooltipContent';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';
import HealthStatus from '../HealthStatus';
import CollectorStatusTotals from './CollectorStatusTotals';
import CollectorUnavailableStatus from './CollectorUnavailableStatus';
import HealthLabelWithDelayed from '../HealthLabelWithDelayed';
import {
    delayedCollectorStatusStyle,
    healthStatusStyles,
    isDelayedSensorHealthStatus,
} from '../../cluster.helpers';
import { ClusterHealthStatus } from '../../clusterTypes';
import HealthStatusNotApplicable from '../HealthStatusNotApplicable';

/*
 * Collector Status in Clusters list if `isList={true}` or Cluster side panel if `isList={false}`
 *
 * Caller is responsible for optional chaining in case healthStatus is null.
 */

type CollectorStatusProps = {
    healthStatus: ClusterHealthStatus;
    isList?: boolean;
};

function CollectorStatus({ healthStatus, isList = false }: CollectorStatusProps): ReactElement {
    if (!healthStatus?.collectorHealthStatus) {
        return <HealthStatusNotApplicable testId="collectorStatus" isList={isList} />;
    }

    const {
        collectorHealthStatus,
        collectorHealthInfo,
        healthInfoComplete,
        sensorHealthStatus,
        lastContact,
    } = healthStatus;
    const isDelayed = !!(lastContact && isDelayedSensorHealthStatus(sensorHealthStatus));
    const { Icon, fgColor } = isDelayed
        ? delayedCollectorStatusStyle
        : healthStatusStyles[collectorHealthStatus];
    const icon = <Icon className={`${isList ? 'inline' : ''} h-4 w-4`} />;
    const currentDatetime = new Date();

    // In rare case that the block does not fit in a narrow column,
    // the space and "whitespace-nowrap" cause time phrase to wrap as a unit.
    // Order arguments according to date-fns@2 convention:
    // If lastContact <= currentDateTime: X units ago
    const statusElement = (
        <HealthLabelWithDelayed
            isDelayed={isDelayed}
            isList={isList}
            clusterHealthItem="collector"
            clusterHealthItemStatus={collectorHealthStatus}
            delayedText={getDistanceStrictAsPhrase(lastContact, currentDatetime)}
        />
    );

    if (collectorHealthInfo) {
        const collectorStatusTotalsElement = (
            <CollectorStatusTotals collectorHealthInfo={collectorHealthInfo} />
        );
        const infoElement = healthInfoComplete ? (
            collectorStatusTotalsElement
        ) : (
            <div>
                {collectorStatusTotalsElement}
                <div data-testid="collectorInfoComplete">
                    <strong>Upgrade Sensor</strong> to get complete Collector health information
                </div>
            </div>
        );

        return isList ? (
            <Tooltip
                content={
                    <DetailedTooltipContent
                        title="Collector Health Information"
                        body={infoElement}
                    />
                }
            >
                <div className="inline">
                    <HealthStatus icon={icon} iconColor={fgColor} isList={isList}>
                        {statusElement}
                    </HealthStatus>
                </div>
            </Tooltip>
        ) : (
            <HealthStatus icon={icon} iconColor={fgColor}>
                <div>
                    {statusElement}
                    {infoElement}
                </div>
            </HealthStatus>
        );
    }

    if (collectorHealthStatus === 'UNAVAILABLE') {
        return (
            <CollectorUnavailableStatus
                isList={isList}
                icon={icon}
                fgColor={fgColor}
                statusElement={statusElement}
            />
        );
    }

    // UNINITIALIZED
    return (
        <HealthStatus icon={icon} iconColor={fgColor} isList={isList}>
            {statusElement}
        </HealthStatus>
    );
}

export default CollectorStatus;
