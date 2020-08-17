import PropTypes from 'prop-types';
import React from 'react';

import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';

import { healthStatusLabels } from 'messages/common';
import { getDateTime, getDistanceStrict } from 'utils/dateUtils';

import HealthStatus from './HealthStatus';
import { healthStatusStyles, isDelayedSensorHealthStatus } from '../cluster.helpers';

/*
 * Sensor Status in Clusters list or Cluster side panel
 *
 * Caller is responsible for optional chaining in case healthStatus is null.
 */
const SensorStatus = ({ sensorHealthStatus, lastContact, currentDatetime }) => {
    if (sensorHealthStatus) {
        const { Icon, bgColor, fgColor } = healthStatusStyles[sensorHealthStatus];
        const labelElement = (
            <span className={`${bgColor} ${fgColor}`}>
                {healthStatusLabels[sensorHealthStatus]}
            </span>
        );

        const isDelayed = lastContact && isDelayedSensorHealthStatus(sensorHealthStatus);

        // In rare case that the block does not fit in a narrow column,
        // the space and "whitespace-no-wrap" cause time phrase to wrap as a unit.
        const statusElement = isDelayed ? (
            <div>
                {labelElement}{' '}
                <span className="whitespace-no-wrap">{`for ${getDistanceStrict(
                    lastContact,
                    currentDatetime
                )}`}</span>
            </div>
        ) : (
            <div>{labelElement}</div>
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

    return null;
};

SensorStatus.propTypes = {
    sensorHealthStatus: PropTypes.oneOf(['UNINITIALIZED', 'UNHEALTHY', 'DEGRADED', 'HEALTHY']),
    lastContact: PropTypes.string, // ISO 8601
    currentDatetime: PropTypes.instanceOf(Date).isRequired,
};

SensorStatus.defaultProps = {
    sensorHealthStatus: null,
    lastContact: null,
};

export default SensorStatus;
