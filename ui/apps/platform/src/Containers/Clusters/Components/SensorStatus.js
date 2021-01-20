import PropTypes from 'prop-types';
import React from 'react';

import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';

import { healthStatusLabels } from 'messages/common';
import { getDateTime, getDistanceStrict } from 'utils/dateUtils';

import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import { healthStatusStyles, isDelayedSensorHealthStatus } from '../cluster.helpers';

const testId = 'sensorStatus';

/*
 * Sensor Status in Clusters list or Cluster side panel
 *
 * Caller is responsible for optional chaining in case healthStatus is null.
 */
const SensorStatus = ({ healthStatus, currentDatetime }) => {
    if (healthStatus?.sensorHealthStatus) {
        const { sensorHealthStatus, lastContact } = healthStatus;
        const { Icon, bgColor, fgColor } = healthStatusStyles[sensorHealthStatus];
        const labelElement = (
            <span className={`${bgColor} ${fgColor}`}>
                {healthStatusLabels[sensorHealthStatus]}
            </span>
        );

        const isDelayed = lastContact && isDelayedSensorHealthStatus(sensorHealthStatus);

        // In rare case that the block does not fit in a narrow column,
        // the space and "whitespace-nowrap" cause time phrase to wrap as a unit.
        const statusElement = isDelayed ? (
            <div data-testid={testId}>
                {labelElement}{' '}
                <span className="whitespace-nowrap">{`for ${getDistanceStrict(
                    lastContact,
                    currentDatetime
                )}`}</span>
            </div>
        ) : (
            <div data-testid={testId}>{labelElement}</div>
        );

        const sensorStatus = (
            <HealthStatus Icon={Icon} iconColor={fgColor}>
                {statusElement}
            </HealthStatus>
        );

        if (lastContact) {
            // Tooltip has absolute time (in ISO 8601 format) to find info from logs.
            return (
                <Tooltip
                    content={
                        <TooltipOverlay>{`Last contact: ${getDateTime(
                            lastContact
                        )}`}</TooltipOverlay>
                    }
                >
                    <div>{sensorStatus}</div>
                </Tooltip>
            );
        }

        return sensorStatus;
    }

    return <HealthStatusNotApplicable testId={testId} />;
};

SensorStatus.propTypes = {
    healthStatus: PropTypes.shape({
        sensorHealthStatus: PropTypes.oneOf(['UNINITIALIZED', 'UNHEALTHY', 'DEGRADED', 'HEALTHY']),
        lastContact: PropTypes.string, // ISO 8601
    }),
    currentDatetime: PropTypes.instanceOf(Date).isRequired,
};

SensorStatus.defaultProps = {
    healthStatus: null,
};

export default SensorStatus;
