import React from 'react';
import PropTypes from 'prop-types';
import { ExternalLink } from 'react-feather';
import { differenceInDays } from 'date-fns';

import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';
import { getDate, getDayOfWeek, getDistanceStrictAsPhrase } from 'utils/dateUtils';

import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import { healthStatusStyles } from '../cluster.helpers';

const diffDegradedMin = 7; // Unhealthy if less than a week in the future
const diffHealthyMin = 30; // Degraded if less than a month in the future

const testId = 'credentialExpiration';

const CredentialExpiration = ({ certExpiryStatus, currentDatetime, isList }) => {
    if (certExpiryStatus?.sensorCertExpiry) {
        const { sensorCertExpiry } = certExpiryStatus;

        // date-fns@2: differenceInDays(parseISO(sensorCertExpiry, currentDatetime))
        const diffInDays = differenceInDays(sensorCertExpiry, currentDatetime);

        // Adapt health status categories to certificate expiration.
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

        /*
         * If status is healthy or expiration is today: distance only
         * If expiration is within the next week: distance and day
         * If expiration is in the past or at least a week in the future: distance and date
         */
        const statusElement =
            healthStatus === 'HEALTHY' || diffInDays === 0 ? (
                <div data-testid={testId}>{distanceElement}</div>
            ) : (
                <div data-testid={testId}>
                    {distanceElement}{' '}
                    <span className="whitespace-no-wrap">{`on ${
                        diffInDays > 0 && diffInDays < 7
                            ? getDayOfWeek(sensorCertExpiry)
                            : getDate(sensorCertExpiry)
                    }`}</span>
                </div>
            );

        if (healthStatus === 'HEALTHY') {
            // A tooltip displays expiration date, which is at least a month in the future.
            return (
                <Tooltip
                    content={
                        <TooltipOverlay>{`Expiration date: ${getDate(
                            sensorCertExpiry
                        )}`}</TooltipOverlay>
                    }
                >
                    <div>
                        <HealthStatus Icon={Icon} iconColor={fgColor}>
                            {statusElement}
                        </HealthStatus>
                    </div>
                </Tooltip>
            );
        }

        // Cluster side panel has external link to heading of topic in Help Center.
        return (
            <HealthStatus Icon={Icon} iconColor={fgColor}>
                {isList ? (
                    statusElement
                ) : (
                    <div>
                        {statusElement}
                        <div className="flex flex-row items-end leading-tight text-tertiary-700">
                            <a
                                href="/docs/product/configure-stackrox/reissue-internal-certificates/#secured-clusters-sensor-collector-admission-controller"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="underline"
                            >
                                Re-issue internal certificates
                            </a>
                            <span className="flex-shrink-0 ml-2">
                                <ExternalLink className="h-4 w-4" />
                            </span>
                        </div>
                    </div>
                )}
            </HealthStatus>
        );
    }

    return <HealthStatusNotApplicable testId={testId} />;
};

CredentialExpiration.propTypes = {
    certExpiryStatus: PropTypes.shape({
        sensorCertExpiry: PropTypes.string, // ISO 8601
    }),
    currentDatetime: PropTypes.instanceOf(Date).isRequired,
    isList: PropTypes.bool.isRequired,
};

CredentialExpiration.defaultProps = {
    certExpiryStatus: null,
};

export default CredentialExpiration;
