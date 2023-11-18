import React from 'react';
import { Link } from 'react-router-dom';
import pluralize from 'pluralize';
import {
    Button,
    Flex,
    FlexItem,
    List,
    ListItem,
    Modal,
    Text,
    TextVariants,
} from '@patternfly/react-core';

import { exceptionManagementPath } from 'routePaths';
import {
    ExceptionExpiry,
    VulnerabilityException,
    VulnerabilityExceptionComment,
    isDeferralException,
    isFalsePositiveException,
} from 'services/VulnerabilityExceptionService';
import { getDate, getDateTime, getDistanceStrictAsPhrase } from 'utils/dateUtils';
import useModal from 'hooks/useModal';

export type RequestContext =
    | 'PENDING_REQUESTS'
    | 'APPROVED_DEFERRALS'
    | 'APPROVED_FALSE_POSITIVES'
    | 'DENIED_REQUESTS';

export type RequestIDLinkProps = {
    id: VulnerabilityException['id'];
    name: VulnerabilityException['name'];
};

export function RequestIDLink({ id, name }: RequestIDLinkProps) {
    return <Link to={`${exceptionManagementPath}/requests/${id}`}>{name}</Link>;
}

export type RequesterProps = {
    requester: VulnerabilityException['requester'];
};
export function Requester({ requester }: RequesterProps) {
    return <div>{requester?.name || '-'}</div>;
}

export type RequestedActionProps = {
    exception: VulnerabilityException;
    context: RequestContext;
};

export function getShouldUseUpdatedRequest(
    exception: VulnerabilityException,
    context: RequestContext
): boolean {
    switch (exception.status) {
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

export function RequestedAction({ exception, context }: RequestedActionProps) {
    return <div>{getRequestedAction(exception, context)}</div>;
}

export type RequestCreatedAtProps = {
    createdAt: VulnerabilityException['createdAt'];
};
export function RequestCreatedAt({ createdAt }: RequestCreatedAtProps) {
    return <div>{getDate(createdAt)}</div>;
}

export type RequestExpiresProps = {
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

export function RequestExpires({ exception, context }: RequestExpiresProps) {
    return <div>{getExpiresDate(exception, context)}</div>;
}

export type RequestScopeProps = {
    scope: VulnerabilityException['scope'];
};

export function RequestScope({ scope }: RequestScopeProps) {
    return (
        <div>{`${scope.imageScope.registry}/${scope.imageScope.remote}:${scope.imageScope.tag}`}</div>
    );
}

export type RequestedItemsProps = {
    exception: VulnerabilityException;
    context: RequestContext;
};

export function RequestedItems({ exception, context }: RequestedItemsProps) {
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

export type RequestCommentProps = {
    comment: VulnerabilityExceptionComment;
};

export function RequestComment({ comment }: RequestCommentProps) {
    return (
        <Flex direction={{ default: 'column' }}>
            <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsSm' }}>
                <Text className="pf-u-font-weight-bold" component={TextVariants.p}>
                    {comment.user.name}
                </Text>
                <Text component={TextVariants.small}>({getDateTime(comment.createdAt)})</Text>
            </Flex>
            <FlexItem>{comment.message}</FlexItem>
        </Flex>
    );
}

export type RequestCommentsProps = {
    comments: VulnerabilityExceptionComment[];
};

export function RequestComments({ comments }: RequestCommentsProps) {
    const { isModalOpen, openModal, closeModal } = useModal();

    return (
        <>
            <Button variant="link" isInline onClick={openModal}>{`${comments.length} ${pluralize(
                'comment',
                comments.length
            )}`}</Button>
            <Modal variant="small" title="Comments" isOpen={isModalOpen} onClose={closeModal}>
                <List isPlain isBordered>
                    {comments.map((comment) => {
                        return (
                            <ListItem key={comment.id}>
                                <RequestComment comment={comment} />
                            </ListItem>
                        );
                    })}
                </List>
            </Modal>
        </>
    );
}
