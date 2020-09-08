import React from 'react';
import PropTypes from 'prop-types';
import { ExternalLink } from 'react-feather';
import { differenceInDays } from 'date-fns';

import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';
import { getDate, getDayOfWeek, getDistanceStrictAsPhrase } from 'utils/dateUtils';

import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import { getCredentialExpirationStatus, healthStatusStyles } from '../cluster.helpers';

const testId = 'credentialExpiration';

const CredentialExpiration = ({ certExpiryStatus, currentDatetime, isList }) => {
    if (certExpiryStatus?.sensorCertExpiry) {
        const { sensorCertExpiry } = certExpiryStatus;

        // Adapt health status categories to certificate expiration.
        const healthStatus = getCredentialExpirationStatus(sensorCertExpiry, currentDatetime);
        const { Icon, bgColor, fgColor } = healthStatusStyles[healthStatus];

        // Order arguments according to date-fns@2 convention:
        // If sensorCertExpiry > currentDateTime: in X units
        // If sensorCertExpiry <= currentDateTime: X units ago
        const distanceElement = (
            <span className={`${bgColor} ${fgColor} whitespace-no-wrap`}>
                {getDistanceStrictAsPhrase(sensorCertExpiry, currentDatetime)}
            </span>
        );

        let expirationElement = null;
        if (healthStatus === 'HEALTHY') {
            // A tooltip displays expiration date, which is at least a month in the future.
            expirationElement = (
                <Tooltip
                    content={
                        <TooltipOverlay>{`Expiration date: ${getDate(
                            sensorCertExpiry
                        )}`}</TooltipOverlay>
                    }
                >
                    <div data-testid={testId}>{distanceElement}</div>
                </Tooltip>
            );
        } else {
            // date-fns@2: differenceInDays(parseISO(sensorCertExpiry, currentDatetime))
            const diffInDays = differenceInDays(sensorCertExpiry, currentDatetime);

            if (diffInDays === 0) {
                expirationElement = <div data-testid={testId}>{distanceElement}</div>;
            } else {
                expirationElement = (
                    <div data-testid={testId}>
                        {distanceElement}{' '}
                        <span className="whitespace-no-wrap">{`on ${
                            diffInDays > 0 && diffInDays < 7
                                ? getDayOfWeek(sensorCertExpiry)
                                : getDate(sensorCertExpiry)
                        }`}</span>
                    </div>
                );
            }
        }

        return (
            <HealthStatus Icon={Icon} iconColor={fgColor}>
                {isList || healthStatus === 'HEALTHY' ? (
                    expirationElement
                ) : (
                    <div>
                        {expirationElement}
                        <div className="flex flex-row items-end leading-tight text-tertiary-700">
                            <a
                                href="/docs/product/docs/configure-stackrox/reissue-internal-certificates/#secured-clusters-sensor-collector-admission-controller"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="underline"
                                data-testid="reissueCertificatesLink"
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
