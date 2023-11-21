import React, { useCallback } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Tab,
    TabTitleText,
    Tabs,
    Title,
    pluralize,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import { exceptionManagementPath } from 'routePaths';
import useSet from 'hooks/useSet';
import useRestQuery from 'hooks/useRestQuery';
import { ensureExhaustive } from 'utils/type.utils';
import {
    VulnerabilityException,
    fetchVulnerabilityExceptionById,
    isDeferralException,
    isFalsePositiveException,
} from 'services/VulnerabilityExceptionService';

import NotFoundMessage from 'Components/NotFoundMessage';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import RequestCVEsTable from './components/RequestCVEsTable';
import TableErrorComponent from '../WorkloadCves/components/TableErrorComponent';
import RequestOverview from './components/RequestOverview';
import useURLStringUnion from 'hooks/useURLStringUnion';

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

function ExceptionRequestDetailsPage() {
    const [selectedContext, setSelectedContext] = useURLStringUnion('context', contextValues);
    const expandedRowSet = useSet<string>();
    const { requestId } = useParams();

    const vulnerabilityExceptionByIdFn = useCallback(
        () => fetchVulnerabilityExceptionById(requestId),
        [requestId]
    );
    const {
        data: vulnerabilityException,
        loading,
        error,
    } = useRestQuery(vulnerabilityExceptionByIdFn);

    function handleTabClick(event, value) {
        setSelectedContext(value);
    }

    if (loading && !vulnerabilityException) {
        return (
            <Bullseye>
                <Spinner isSVG />
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
                message={`A request with ID ${requestId as string} could not be found.`}
            />
        );
    }

    const { status, cves, scope } = vulnerabilityException;
    const isApprovedPendingUpdate = status === 'APPROVED_PENDING_UPDATE';
    const relevantCVEs =
        selectedContext === 'CURRENT' ? cves : getCVEsForUpdatedRequest(vulnerabilityException);

    return (
        <>
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={exceptionManagementPath}>
                        Exception management
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{vulnerabilityException.name}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Flex direction={{ default: 'column' }}>
                    <Title headingLevel="h1">Request {vulnerabilityException.name}</Title>
                    <FlexItem>{getSubtitleText(vulnerabilityException)}</FlexItem>
                </Flex>
            </PageSection>
            <PageSection className="pf-u-p-0">
                {isApprovedPendingUpdate && (
                    <Tabs
                        activeKey={selectedContext}
                        onSelect={handleTabClick}
                        component="nav"
                        className="pf-u-pl-lg pf-u-background-color-100"
                    >
                        <Tab
                            eventKey="PENDING_UPDATE"
                            title={<TabTitleText>Requested update</TabTitleText>}
                        />
                        <Tab
                            eventKey="CURRENT"
                            title={<TabTitleText>Latest approved</TabTitleText>}
                        />
                    </Tabs>
                )}

                <PageSection>
                    <RequestOverview exception={vulnerabilityException} context={selectedContext} />
                </PageSection>
                <PageSection>
                    <RequestCVEsTable
                        cves={relevantCVEs}
                        scope={scope}
                        expandedRowSet={expandedRowSet}
                    />
                </PageSection>
            </PageSection>
        </>
    );
}

export default ExceptionRequestDetailsPage;
