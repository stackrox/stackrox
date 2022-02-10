import React, { ReactElement } from 'react';
import { ExternalLink } from 'react-feather';
import { differenceInDays } from 'date-fns';

import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';
import { getDate, getDayOfWeek, getDistanceStrictAsPhrase } from 'utils/dateUtils';

import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import { getCredentialExpirationStatus, healthStatusStyles } from '../cluster.helpers';

const testId = 'credentialExpiration';

type CredentialExpirationProps = {
    certExpiryStatus?: {
        sensorCertExpiry: string; // ISO 8601
    };
    isList?: boolean;
};

function CredentialExpiration({
    certExpiryStatus,
    isList = false,
}: CredentialExpirationProps): ReactElement {
    if (!certExpiryStatus?.sensorCertExpiry) {
        return <HealthStatusNotApplicable testId={testId} />;
    }

    const { sensorCertExpiry } = certExpiryStatus;
    const currentDatetime = new Date();

    // Adapt health status categories to certificate expiration.
    const healthStatus = getCredentialExpirationStatus(sensorCertExpiry, currentDatetime);
    const { Icon, bgColor, fgColor } = healthStatusStyles[healthStatus];
    const icon = <Icon className="h-4 w-4" />;

    // Order arguments according to date-fns@2 convention:
    // If sensorCertExpiry > currentDateTime: in X units
    // If sensorCertExpiry <= currentDateTime: X units ago
    const distanceElement = (
        <span className={`${bgColor} ${fgColor} whitespace-nowrap`}>
            {getDistanceStrictAsPhrase(sensorCertExpiry, currentDatetime)}
        </span>
    );

    let expirationElement = <></>;
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
                    <span className="whitespace-nowrap">{`on ${
                        diffInDays > 0 && diffInDays < 7
                            ? getDayOfWeek(sensorCertExpiry)
                            : getDate(sensorCertExpiry)
                    }`}</span>
                </div>
            );
        }
    }

    return (
        <HealthStatus icon={icon} iconColor={fgColor}>
            {isList || healthStatus === 'HEALTHY' ? (
                expirationElement
            ) : (
                <div>
                    {expirationElement}
                    <div className="flex flex-row items-end leading-tight text-tertiary-700">
                        <a
                            href="/docs/product/rhacs/latest/configuration/reissue-internal-certificates.html#reissue-internal-certificates-secured-clusters"
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

export default CredentialExpiration;
