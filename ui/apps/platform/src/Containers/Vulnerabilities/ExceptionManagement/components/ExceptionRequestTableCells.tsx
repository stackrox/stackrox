import React from 'react';
import { Link } from 'react-router-dom';
import pluralize from 'pluralize';

import { exceptionManagementPath } from 'routePaths';
import {
    ExceptionExpiry,
    VulnerabilityDeferralException,
    VulnerabilityException,
    isDeferralException,
    isFalsePositiveException,
} from 'services/VulnerabilityExceptionService';
import { getDate, getDistanceStrictAsPhrase } from 'utils/dateUtils';

// @TODO: Add tests for these

export type RequestContext = 'PENDING_REQUESTS' | 'APPROVED_DEFERRALS';

export type RequestIDTableCellProps = {
    id: VulnerabilityException['id'];
    name: VulnerabilityException['name'];
};

export function RequestIDTableCell({ id, name }: RequestIDTableCellProps) {
    return <Link to={`${exceptionManagementPath}/requests/${id}`}>{name}</Link>;
}

export type RequesterTableCellProps = {
    requester: VulnerabilityException['requester'];
};
export function RequesterTableCell({ requester }: RequesterTableCellProps) {
    return <div>{requester.name}</div>;
}

export type RequestedActionTableCellProps = {
    exception: VulnerabilityException;
    context: RequestContext;
};

export function getDeferralExpiryToUse(
    exception: VulnerabilityDeferralException,
    context: RequestContext
): ExceptionExpiry {
    switch (exception.exceptionStatus) {
        case 'PENDING':
            return exception.deferralReq.expiry;
        case 'APPROVED':
        case 'DENIED':
            if (exception.deferralUpdate) {
                return exception.deferralUpdate.expiry;
            }
            return exception.deferralReq.expiry;
        case 'APPROVED_PENDING_UPDATE':
            if (context === 'PENDING_REQUESTS' && exception.deferralUpdate) {
                return exception.deferralUpdate.expiry;
            }
            return exception.deferralReq.expiry;
        default:
            return exception.deferralReq.expiry;
    }
}

export function getRequestedAction(
    exception: VulnerabilityException,
    context: RequestContext
): string {
    if (isDeferralException(exception)) {
        const exceptionExpiry: ExceptionExpiry = getDeferralExpiryToUse(exception, context);
        let duration = 'indefinitely';
        if (exceptionExpiry.expiryType === 'TIME' && exceptionExpiry.expiresOn) {
            duration = getDistanceStrictAsPhrase(
                exceptionExpiry.expiresOn,
                exception.lastUpdated,
                'd'
            );
        } else if (exceptionExpiry.expiryType === 'ALL_CVE_FIXABLE') {
            duration = 'when all fixed';
        } else if (exceptionExpiry.expiryType === 'ANY_CVE_FIXABLE') {
            duration = 'when any fixed';
        }
        return `Deferral (${duration})`;
    }
    if (isFalsePositiveException(exception)) {
        return 'False positive';
    }
    return '-';
}

export function RequestedActionTableCell({ exception, context }: RequestedActionTableCellProps) {
    return <div>{getRequestedAction(exception, context)}</div>;
}

export type RequestedTableCellProps = {
    createdAt: VulnerabilityException['createdAt'];
};
export function RequestedTableCell({ createdAt }: RequestedTableCellProps) {
    return <div>{getDate(createdAt)}</div>;
}

export type ExpiresTableCellProps = {
    exception: VulnerabilityException;
    context: RequestContext;
};

export function getExpiresDate(exception: VulnerabilityException, context: RequestContext): string {
    if (isDeferralException(exception)) {
        const exceptionExpiry: ExceptionExpiry = getDeferralExpiryToUse(exception, context);
        if (exceptionExpiry.expiryType === 'TIME' && exceptionExpiry.expiresOn) {
            return getDate(exceptionExpiry.expiresOn);
        }
    }
    return '-';
}

export function ExpiresTableCell({ exception, context }: ExpiresTableCellProps) {
    return <div>{getExpiresDate(exception, context)}</div>;
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
