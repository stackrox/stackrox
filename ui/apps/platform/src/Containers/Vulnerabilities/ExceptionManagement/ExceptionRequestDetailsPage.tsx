import React, { useState, useCallback } from 'react';
import {
    Alert,
    AlertActionCloseButton,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Tab,
    TabContent,
    TabTitleText,
    Tabs,
    Title,
    pluralize,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import { exceptionManagementPath } from 'routePaths';
import useSet from 'hooks/useSet';
import useRestQuery from 'hooks/useRestQuery';
import usePermissions from 'hooks/usePermissions';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useAuthStatus from 'hooks/useAuthStatus';
import { ensureExhaustive } from 'utils/type.utils';
import {
    VulnerabilityException,
    fetchVulnerabilityExceptionById,
    isDeferralException,
    isFalsePositiveException,
} from 'services/VulnerabilityExceptionService';

import PageTitle from 'Components/PageTitle';
import NotFoundMessage from 'Components/NotFoundMessage';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';

import RequestCVEsTable from './components/RequestCVEsTable';
import RequestOverview from './components/RequestOverview';
import RequestApprovalButtonModal from './components/RequestApprovalButtonModal';
import RequestDenialButtonModal from './components/RequestDenialButtonModal';
import RequestCancelButtonModal from './components/RequestCancelButtonModal';
import RequestUpdateButtonModal from './components/RequestUpdateButtonModal';
import { getVulnerabilityState } from './utils';

import './ExceptionRequestDetailsPage.css';

export const contextValues = ['CURRENT', 'PENDING_UPDATE'] as const;

function getSubtitleText(exception: VulnerabilityException) {
    const numCVEs = `${pluralize(exception.cves.length, 'CVE')}`;
    switch (exception.status) {
        case 'PENDING':
            return `Pending (${numCVEs})`;
        case 'DENIED':
            return `Denied (${numCVEs})`;
        case 'APPROVED':
            return `Approved (${numCVEs})`;
        case 'APPROVED_PENDING_UPDATE':
            return `Approved, pending update`;
        default:
            return ensureExhaustive(exception.status);
    }
}

export function getCVEsForUpdatedRequest(exception: VulnerabilityException): string[] {
    if (isDeferralException(exception) && exception.deferralUpdate) {
        return exception.deferralUpdate.cves;
    }
    if (isFalsePositiveException(exception) && exception.falsePositiveUpdate) {
        return exception.falsePositiveUpdate.cves;
    }
    return exception.cves;
}

const tabContentId = 'ExceptionRequestDetails';

function ExceptionRequestDetailsPage() {
    const { requestId } = useParams() as { requestId: string };
    const { hasReadWriteAccess } = usePermissions();
    const { currentUser } = useAuthStatus();
    const hasWriteAccessForApproving = hasReadWriteAccess('VulnerabilityManagementApprovals');

    const [selectedContext, setSelectedContext] = useURLStringUnion('context', contextValues);
    const expandedRowSet = useSet<string>();
    const [successMessage, setSuccessMessage] = useState<string | null>(null);

    const vulnerabilityExceptionByIdFn = useCallback(
        () => fetchVulnerabilityExceptionById(requestId),
        [requestId]
    );
    const {
        data: vulnerabilityException,
        isLoading,
        error,
        refetch,
    } = useRestQuery(vulnerabilityExceptionByIdFn);

    function handleTabClick(event, value) {
        setSelectedContext(value);
    }

    function onApprovalSuccess() {
        refetch();
        setSuccessMessage(`The vulnerability request was successfully approved.`);
    }

    function onDenialSuccess() {
        refetch();
        setSuccessMessage(`The vulnerability request was successfully denied.`);
    }

    function onCancelSuccess() {
        refetch();
        setSuccessMessage(`The vulnerability request was successfully canceled.`);
    }

    function onUpdateSuccess() {
        refetch();
        setSuccessMessage(`The vulnerability request was successfully updated.`);
    }

    if (isLoading && !vulnerabilityException) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <PageSection variant="light">
                <TableErrorComponent
                    error={error}
                    message="An error occurred. Try refreshing again"
                />
            </PageSection>
        );
    }

    if (!vulnerabilityException) {
        return (
            <NotFoundMessage
                title="404: We couldn't find that page"
                message={`A request with ID ${requestId} could not be found.`}
            />
        );
    }

    const { status, cves, scope, requester, expired } = vulnerabilityException;

    const isApprovedPendingUpdate = status === 'APPROVED_PENDING_UPDATE';
    const showApproveDenyButtons =
        hasWriteAccessForApproving &&
        !expired &&
        (status === 'PENDING' || status === 'APPROVED_PENDING_UPDATE');
    const showCancelButton =
        !expired && currentUser.userId === requester?.id && status !== 'DENIED';
    const showUpdateButton =
        !expired &&
        currentUser.userId === requester?.id &&
        (status === 'PENDING' || status === 'APPROVED' || status === 'APPROVED_PENDING_UPDATE');

    const relevantCVEs =
        selectedContext === 'CURRENT' ? cves : getCVEsForUpdatedRequest(vulnerabilityException);

    const vulnerabilityState = getVulnerabilityState(vulnerabilityException);

    return (
        <>
            <PageTitle title="Exception Management - Request Details" />
            {successMessage && (
                <Alert
                    variant="success"
                    isInline
                    title={successMessage}
                    component="p"
                    actionClose={<AlertActionCloseButton onClose={() => setSuccessMessage(null)} />}
                />
            )}
            {expired && (
                <Alert variant="warning" isInline title="Request Canceled." component="p">
                    You are viewing a canceled request. If this cancellation was not intended,
                    please submit a new request
                </Alert>
            )}
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={exceptionManagementPath}>
                        Exception management
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{vulnerabilityException.name}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Flex>
                    <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">Request {vulnerabilityException.name}</Title>
                        <FlexItem>{getSubtitleText(vulnerabilityException)}</FlexItem>
                    </Flex>
                    {showCancelButton && (
                        <RequestCancelButtonModal
                            exception={vulnerabilityException}
                            onSuccess={onCancelSuccess}
                        />
                    )}
                    {showApproveDenyButtons && (
                        <RequestDenialButtonModal
                            exception={vulnerabilityException}
                            onSuccess={onDenialSuccess}
                        />
                    )}
                    {showApproveDenyButtons && (
                        <RequestApprovalButtonModal
                            exception={vulnerabilityException}
                            onSuccess={onApprovalSuccess}
                        />
                    )}
                    {showUpdateButton && (
                        <RequestUpdateButtonModal
                            exception={vulnerabilityException}
                            onSuccess={onUpdateSuccess}
                        />
                    )}
                </Flex>
            </PageSection>
            <PageSection className="pf-v5-u-p-0">
                {isApprovedPendingUpdate && (
                    <Tabs
                        activeKey={selectedContext}
                        onSelect={handleTabClick}
                        className="pf-v5-u-pl-lg pf-v5-u-background-color-100"
                    >
                        <Tab
                            eventKey="PENDING_UPDATE"
                            tabContentId={tabContentId}
                            title={<TabTitleText>Requested update</TabTitleText>}
                        />
                        <Tab
                            eventKey="CURRENT"
                            tabContentId={tabContentId}
                            title={<TabTitleText>Latest approved</TabTitleText>}
                        />
                    </Tabs>
                )}
                <TabContent id={tabContentId}>
                    <PageSection>
                        <RequestOverview
                            exception={vulnerabilityException}
                            context={selectedContext}
                        />
                    </PageSection>
                    <PageSection>
                        <RequestCVEsTable
                            cves={relevantCVEs}
                            scope={scope}
                            expandedRowSet={expandedRowSet}
                            vulnerabilityState={vulnerabilityState}
                        />
                    </PageSection>
                </TabContent>
            </PageSection>
        </>
    );
}

export default ExceptionRequestDetailsPage;
