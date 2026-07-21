import type { ReactElement } from 'react';
import { differenceInDays } from 'date-fns';
import { Tooltip } from '@patternfly/react-core';

import { getDate, getDayOfWeek, getDistanceStrictAsPhrase, getTime } from 'utils/dateUtils';
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useMetadata from 'hooks/useMetadata';
import { getVersionedDocs } from 'utils/versioning';
import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import { getCredentialExpirationStatus, healthStatusStylesLegacy } from '../cluster.helpers';
import type { CertExpiryStatus } from '../clusterTypes';

const testId = 'credentialExpiration';

type CredentialExpirationProps = {
    certExpiryStatus?: CertExpiryStatus;
    autoRefreshEnabled: boolean;
    isList?: boolean;
};

function CredentialExpiration({
    certExpiryStatus,
    autoRefreshEnabled,
    isList = false,
}: CredentialExpirationProps): ReactElement {
    const { version } = useMetadata();

    if (!certExpiryStatus?.sensorCertExpiry) {
        return <HealthStatusNotApplicable testId={testId} />;
    }

    const { sensorCertExpiry, sensorCertNotBefore, lastRefreshTime, lastRefreshedCertExpiry } =
        certExpiryStatus;
    const currentDatetime = new Date();

    // Adapt health status categories to certificate expiration.
    const healthStatus = getCredentialExpirationStatus(certExpiryStatus, currentDatetime);
    const { Icon, fgColor } = healthStatusStylesLegacy[healthStatus];
    const icon = <Icon className="h-4 w-4" />;

    // When a refresh has occurred after the current connection was established, show the
    // refreshed cert's expiry instead of the (potentially stale) connection cert expiry.
    const refreshedAfterConnection =
        lastRefreshedCertExpiry &&
        lastRefreshTime &&
        sensorCertNotBefore &&
        new Date(sensorCertNotBefore) < new Date(lastRefreshTime);
    const displayedExpiry = refreshedAfterConnection ? lastRefreshedCertExpiry : sensorCertExpiry;

    // Order arguments according to date-fns@2 convention:
    // If expiry > currentDateTime: in X units
    // If expiry <= currentDateTime: X units ago
    const distanceElement = (
        <span className="whitespace-nowrap">
            {getDistanceStrictAsPhrase(displayedExpiry, currentDatetime)}
        </span>
    );

    let expirationElement = <></>;
    const diffInDays = differenceInDays(displayedExpiry, currentDatetime);
    if (healthStatus === 'HEALTHY') {
        let tooltipText: string;
        if (diffInDays === 0) {
            tooltipText = `Expiration time: ${getTime(displayedExpiry)}`;
        } else {
            tooltipText = `Expiration date: ${getDate(displayedExpiry)}`;
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
                        ? getDayOfWeek(displayedExpiry)
                        : getDate(displayedExpiry)
                }`}</span>
            </div>
        );
    }

    return (
        <HealthStatus icon={icon} iconColor={fgColor}>
            {isList || healthStatus === 'HEALTHY' ? (
                <div>
                    {expirationElement}
                    {autoRefreshEnabled && <div>Auto-refresh enabled</div>}
                </div>
            ) : (
                <div>
                    {expirationElement}
                    {version && (
                        <div className="flex flex-row items-end leading-tight">
                            <ExternalLink>
                                <a
                                    href={getVersionedDocs(
                                        version,
                                        'configuring/reissue-internal-certificates#reissue-internal-certificates-secured-clusters_reissue-internal-certificates'
                                    )}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    data-testid="reissueCertificatesLink"
                                >
                                    Re-issue internal certificates
                                </a>
                            </ExternalLink>
                        </div>
                    )}
                </div>
            )}
        </HealthStatus>
    );
}

export default CredentialExpiration;
