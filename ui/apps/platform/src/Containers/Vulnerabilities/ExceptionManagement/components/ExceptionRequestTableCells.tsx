import React from 'react';
import { Link } from 'react-router-dom';
import pluralize from 'pluralize';

import { exceptionManagementPath } from 'routePaths';
import {
    ExceptionExpiry,
    VulnerabilityException,
    isDeferralException,
    isFalsePositiveException,
} from 'services/VulnerabilityExceptionService';
import { getDate, getDistanceStrictAsPhrase } from 'utils/dateUtils';

// @TODO: Add tests for these

export type RequestContext = 'PENDING_REQUESTS' | 'APPROVED_DEFERRALS' | 'APPROVED_FALSE_POSITIVES';

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

export function getShouldUseUpdatedRequest(
    exception: VulnerabilityException,
    context: RequestContext
): boolean {
    switch (exception.exceptionStatus) {
        case 'APPROVED_PENDING_UPDATE':
            if (context === 'PENDING_REQUESTS') {
                if (isDeferralException(exception) && exception.deferralUpdate) {
                    return true;
                }
                if (isFalsePositiveException(exception) && exception.falsePositiveUpdate) {
                    return true;
                }
            }
            return false;
        default:
            return false;
    }
}

export function getRequestedAction(
    exception: VulnerabilityException,
    context: RequestContext
): string {
    const shouldUseUpdatedRequest = getShouldUseUpdatedRequest(exception, context);
    if (isDeferralException(exception)) {
        const exceptionExpiry: ExceptionExpiry =
            shouldUseUpdatedRequest && exception.deferralUpdate
                ? exception.deferralUpdate.expiry
                : exception.deferralRequest.expiry;
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
        const deferralText = shouldUseUpdatedRequest ? 'Deferral pending update' : 'Deferred';
        return `${deferralText} (${duration})`;
    }
    if (isFalsePositiveException(exception)) {
        const falsePositiveText = shouldUseUpdatedRequest
            ? 'False positive pending update'
            : 'False positive';
        return falsePositiveText;
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
        const shouldUseUpdatedRequest = getShouldUseUpdatedRequest(exception, context);
        const exceptionExpiry: ExceptionExpiry =
            shouldUseUpdatedRequest && exception.deferralUpdate
                ? exception.deferralUpdate.expiry
                : exception.deferralRequest.expiry;
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
    exception: VulnerabilityException;
    context: RequestContext;
};

export function RequestedItemsTableCell({ exception, context }: RequestedItemsTableCellProps) {
    const shouldUseUpdatedRequest = getShouldUseUpdatedRequest(exception, context);
    let cvesCount = exception.cves.length;
    if (isDeferralException(exception) && shouldUseUpdatedRequest && exception.deferralUpdate) {
        cvesCount = exception.deferralUpdate.cves.length;
    } else if (
        isFalsePositiveException(exception) &&
        shouldUseUpdatedRequest &&
        exception.falsePositiveUpdate
    ) {
        cvesCount = exception.falsePositiveUpdate.cves.length;
    }
    return <div>{`${cvesCount} ${pluralize('CVE', cvesCount)}`}</div>;
}
