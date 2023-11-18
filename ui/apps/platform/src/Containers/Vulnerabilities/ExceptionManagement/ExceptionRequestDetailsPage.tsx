import React, { useCallback } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
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
} from 'services/VulnerabilityExceptionService';

import NotFoundMessage from 'Components/NotFoundMessage';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import {
    RequestExpires,
    RequestedAction,
    RequestCreatedAt,
    RequestScope,
    RequestComments,
    RequestComment,
} from './components/ExceptionRequestTableCells';
import RequestCVEsTable from './components/RequestCVEsTable';
import TableErrorComponent from '../WorkloadCves/components/TableErrorComponent';

import './ExceptionRequestDetailsPage.css';

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
            return `Approved pending update (${numCVEs})`;
        default:
            return ensureExhaustive(exception.status);
    }
}

function ExceptionRequestDetailsPage() {
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

    // There will always be at least 1 comment
    const latestComment =
        vulnerabilityException.comments[vulnerabilityException.comments.length - 1];

    // @TODO: Will need to figure out a good way to distinguish the original request cves from the updated one
    const { cves, scope } = vulnerabilityException;

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
            <PageSection>
                <PageSection variant="light">
                    <Flex direction={{ default: 'column' }}>
                        <Title headingLevel="h2">Overview</Title>
                        <DescriptionList className="vulnerability-exception-request-overview">
                            <DescriptionListGroup>
                                <DescriptionListTerm>Requestor</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {vulnerabilityException.requester?.name || '-'}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Requested action</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <RequestedAction
                                        exception={vulnerabilityException}
                                        context="PENDING_REQUESTS" // @TODO: We need a smarter way to distinguish original vs. updated values here
                                    />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Requested</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <RequestCreatedAt
                                        createdAt={vulnerabilityException.createdAt}
                                    />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Expires</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <RequestExpires
                                        exception={vulnerabilityException}
                                        context="PENDING_REQUESTS" // @TODO: We need a smarter way to distinguish original vs. updated values here
                                    />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Scope</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <RequestScope scope={vulnerabilityException.scope} />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Comments</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <RequestComments comments={vulnerabilityException.comments} />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Latest comment</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <RequestComment comment={latestComment} />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </Flex>
                </PageSection>
            </PageSection>
            <PageSection>
                {/* @TODO: Consider reusability with the Workload CVEs CVE table */}
                <RequestCVEsTable cves={cves} scope={scope} expandedRowSet={expandedRowSet} />
            </PageSection>
        </>
    );
}

export default ExceptionRequestDetailsPage;
