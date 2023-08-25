import React, { ReactElement } from 'react';
import { ExternalLink } from 'react-feather';
import { differenceInDays } from 'date-fns';
import { Tooltip } from '@patternfly/react-core';

import { getTime, getDate, getDayOfWeek, getDistanceStrictAsPhrase } from 'utils/dateUtils';

import useMetadata from 'hooks/useMetadata';
import { getVersionedDocs } from 'utils/versioning';
import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import { getCredentialExpirationStatus, healthStatusStyles } from '../cluster.helpers';
import { CertExpiryStatus } from '../clusterTypes';

const testId = 'credentialExpiration';

type CredentialExpirationProps = {
    certExpiryStatus?: CertExpiryStatus;
    isList?: boolean;
};

function CredentialExpiration({
    certExpiryStatus,
    isList = false,
}: CredentialExpirationProps): ReactElement {
    const { version } = useMetadata();

    if (!certExpiryStatus?.sensorCertExpiry) {
        return <HealthStatusNotApplicable testId={testId} />;
    }

    const { sensorCertExpiry } = certExpiryStatus;
    const currentDatetime = new Date();

    // Adapt health status categories to certificate expiration.
    const healthStatus = getCredentialExpirationStatus(certExpiryStatus, currentDatetime);
    const { Icon, fgColor } = healthStatusStyles[healthStatus];
    const icon = <Icon className="h-4 w-4" />;

    // Order arguments according to date-fns@2 convention:
    // If sensorCertExpiry > currentDateTime: in X units
    // If sensorCertExpiry <= currentDateTime: X units ago
    const distanceElement = (
        <span className="whitespace-nowrap">
            {getDistanceStrictAsPhrase(sensorCertExpiry, currentDatetime)}
        </span>
    );

    let expirationElement = <></>;
    const diffInDays = differenceInDays(sensorCertExpiry, currentDatetime);
    if (healthStatus === 'HEALTHY') {
        let tooltipText: string;
        if (diffInDays === 0) {
            tooltipText = `Expiration time: ${getTime(sensorCertExpiry)}`;
        } else {
            tooltipText = `Expiration date: ${getDate(sensorCertExpiry)}`;
        }
        // A tooltip displays expiration date or time
        expirationElement = (
            <Tooltip content={tooltipText}>
                <div data-testid={testId}>{distanceElement}</div>
            </Tooltip>
        );
    } else if (diffInDays === 0) {
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

    return (
        <HealthStatus icon={icon} iconColor={fgColor}>
            {isList || healthStatus === 'HEALTHY' ? (
                expirationElement
            ) : (
                <div>
                    {expirationElement}
                    {version && (
                        <div className="flex flex-row items-end leading-tight text-tertiary-700">
                            <a
                                href={getVersionedDocs(
                                    version,
                                    'configuration/reissue-internal-certificates.html#reissue-internal-certificates-secured-clusters'
                                )}
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
                    )}
                </div>
            )}
        </HealthStatus>
    );
}

export default CredentialExpiration;
