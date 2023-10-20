import React from 'react';
import { Link } from 'react-router-dom';
import pluralize from 'pluralize';

import { exceptionManagementPath } from 'routePaths';
import {
    VulnerabilityDeferralException,
    VulnerabilityException,
    VulnerabilityFalsePositiveException,
} from 'services/VulnerabilityExceptionService';
import { getDate, getDistanceStrictAsPhrase } from 'utils/dateUtils';

// @TODO: Add tests for these

function isVulnerabilityDeferralException(
    exception: VulnerabilityException
): exception is VulnerabilityDeferralException {
    return exception.targetState === 'DEFERRED';
}

function isVulnerabilityFalsePositiveException(
    exception: VulnerabilityException
): exception is VulnerabilityFalsePositiveException {
    return exception.targetState === 'FALSE_POSITIVE';
}

export type RequestIDTableCellProps = {
    id: VulnerabilityException['id'];
    name: VulnerabilityException['name'];
};

export function RequestIDTableCell({ id, name }: RequestIDTableCellProps) {
    return <Link to={`${exceptionManagementPath}/requests/:${id}`}>{name}</Link>;
}

export type RequesterTableCellProps = {
    requester: VulnerabilityException['requester'];
};
export function RequesterTableCell({ requester }: RequesterTableCellProps) {
    return <div>{requester.name}</div>;
}

export type RequestedActionTableCellProps = {
    exception: VulnerabilityException;
};

function getRequestedAction(exception: VulnerabilityException): string {
    if (isVulnerabilityDeferralException(exception)) {
        const latestExpiry = exception.deferralUpdate
            ? exception.deferralUpdate.expiry
            : exception.deferralReq.expiry;
        let duration = 'indefinitely';
        if (latestExpiry.expiryType === 'TIME' && latestExpiry.expiresOn) {
            duration = getDistanceStrictAsPhrase(
                latestExpiry.expiresOn,
                exception.lastUpdated,
                'd'
            );
        } else if (latestExpiry.expiryType === 'ALL_CVE_FIXABLE') {
            duration = 'when all fixed';
        } else if (latestExpiry.expiryType === 'ANY_CVE_FIXABLE') {
            duration = 'when any fixed';
        }
        return `Deferral (${duration})`;
    }
    if (isVulnerabilityFalsePositiveException(exception)) {
        return 'False positive';
    }
    return '-';
}

export function RequestedActionTableCell({ exception }: RequestedActionTableCellProps) {
    return <div>{getRequestedAction(exception)}</div>;
}

export type RequestedTableCellProps = {
    createdAt: VulnerabilityException['createdAt'];
};
export function RequestedTableCell({ createdAt }: RequestedTableCellProps) {
    return <div>{getDate(createdAt)}</div>;
}

export type ExpiresTableCellProps = {
    exception: VulnerabilityException;
};

function getExpiresDate(exception: VulnerabilityException): string {
    if (isVulnerabilityDeferralException(exception)) {
        const latestExpiry = exception.deferralUpdate
            ? exception.deferralUpdate.expiry
            : exception.deferralReq.expiry;
        if (latestExpiry.expiryType === 'TIME' && latestExpiry.expiresOn) {
            return getDate(latestExpiry.expiresOn);
        }
    }
    return '-';
}

export function ExpiresTableCell({ exception }: ExpiresTableCellProps) {
    return <div>{getExpiresDate(exception)}</div>;
}

export type ScopeTableCellProps = {
    scope: VulnerabilityException['scope'];
};

export function ScopeTableCell({ scope }: ScopeTableCellProps) {
    return (
        <div>{`${scope.imageScope.registry}/${scope.imageScope.remote}:${scope.imageScope.tag}`}</div>
    );
}

export type RequestedItemsTableCellProps = {
    cves: VulnerabilityException['cves'];
};

export function RequestedItemsTableCell({ cves }: RequestedItemsTableCellProps) {
    return <div>{`${cves.length} ${pluralize('CVE', cves.length)}`}</div>;
}
