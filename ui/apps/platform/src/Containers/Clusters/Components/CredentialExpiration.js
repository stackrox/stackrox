import React from 'react';
import PropTypes from 'prop-types';
import { differenceInDays } from 'date-fns';

import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';
import { getDate, getDayOfWeek, getDistanceStrictAsPhrase } from 'utils/dateUtils';

import HealthStatus from './HealthStatus';
import { healthStatusStyles } from '../cluster.helpers';

const diffDegradedMin = 7; // if less, display a day of the week for Unhealthy
const diffHealthyMin = 30; // if less, display a date for Degraded

const CredentialExpiration = ({ certExpiryStatus, currentDatetime }) => {
    if (certExpiryStatus?.sensorCertExpiry) {
        const { sensorCertExpiry } = certExpiryStatus;

        // date-fns@2: differenceInDays(parseISO(sensorCertExpiry, currentDatetime))
        const diffInDays = differenceInDays(sensorCertExpiry, currentDatetime);

        // Adapt health status categories that seem relevant to certificate expiration.
        let healthStatus = 'HEALTHY';
        if (diffInDays < diffDegradedMin) {
            healthStatus = 'UNHEALTHY';
        } else if (diffInDays < diffHealthyMin) {
            healthStatus = 'DEGRADED';
        }

        const { Icon, bgColor, fgColor } = healthStatusStyles[healthStatus];

        // Order arguments according to date-fns@2 convention:
        // If sensorCertExpiry > currentDateTime: in X units
        // If sensorCertExpiry <= currentDateTime: X units ago
        const distanceElement = (
            <span className={`${bgColor} ${fgColor} whitespace-no-wrap`}>
                {getDistanceStrictAsPhrase(sensorCertExpiry, currentDatetime)}
            </span>
        );

        // Display day or date unless expiration is today or more than 1 month in the future.
        const expirationElement =
            diffInDays !== 0 && diffInDays < 60 ? (
                <div>
                    {distanceElement}{' '}
                    <span className="whitespace-no-wrap">{`on ${
                        diffInDays > 0 && diffInDays < 7
                            ? getDayOfWeek(sensorCertExpiry)
                            : getDate(sensorCertExpiry)
                    }`}</span>
                </div>
            ) : (
                <div>{distanceElement}</div>
            );

        // Tooltip has the absolute date (in ISO 8601 format).
        return (
            <Tooltip
                content={<TooltipOverlay>{`Expiration date: ${sensorCertExpiry}`}</TooltipOverlay>}
            >
                <div>
                    <HealthStatus Icon={Icon} iconColor={fgColor}>
                        {expirationElement}
                    </HealthStatus>
                </div>
            </Tooltip>
        );
    }

    return null;
};

CredentialExpiration.propTypes = {
    certExpiryStatus: PropTypes.shape({
        sensorCertExpiry: PropTypes.string, // ISO 8601
    }),
    currentDatetime: PropTypes.instanceOf(Date).isRequired,
};

CredentialExpiration.defaultProps = {
    certExpiryStatus: null,
};

export default CredentialExpiration;
