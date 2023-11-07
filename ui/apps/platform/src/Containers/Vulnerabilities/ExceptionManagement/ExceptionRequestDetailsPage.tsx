import React from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Text,
    TextVariants,
    Title,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import NotFoundMessage from 'Components/NotFoundMessage';
import { VulnerabilityException } from 'services/VulnerabilityExceptionService';
import { getDateTime } from 'utils/dateUtils';
import { exceptionManagementPath } from 'routePaths';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { ensureExhaustive } from 'utils/type.utils';
import { vulnerabilityExceptions } from './mockUtils';
import {
    RequestExpires,
    RequestedAction,
    RequestCreatedAt,
    RequestScope,
} from './components/ExceptionRequestTableCells';

import './ExceptionRequestDetailsPage.css';

function getSubtitleText(exception: VulnerabilityException) {
    switch (exception.exceptionStatus) {
        case 'PENDING':
            return 'Pending';
        case 'DENIED':
            return `Denied (${exception.cves.length} CVEs)`;
        case 'APPROVED':
            return `Approved (${exception.cves.length} CVEs)`;
        case 'APPROVED_PENDING_UPDATE':
            return `Approved pending update (${exception.cves.length} CVEs)`;
        default:
            return ensureExhaustive(exception.exceptionStatus);
    }
}

function ExceptionRequestDetailsPage() {
    const { requestId } = useParams();
    const vulnerabilityException = vulnerabilityExceptions.find(
        (exception) => exception.id === requestId
    );

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
                                    {vulnerabilityException.requester.name}
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
                                    {vulnerabilityException.comments.length}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Latest comment</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <Flex direction={{ default: 'column' }}>
                                        <Flex
                                            direction={{ default: 'row' }}
                                            spaceItems={{ default: 'spaceItemsSm' }}
                                        >
                                            <Text
                                                className="pf-u-font-weight-bold"
                                                component={TextVariants.p}
                                            >
                                                {latestComment.user.name}
                                            </Text>
                                            <Text component={TextVariants.small}>
                                                ({getDateTime(latestComment.createdAt)})
                                            </Text>
                                        </Flex>
                                        <FlexItem>{latestComment.message}</FlexItem>
                                    </Flex>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </Flex>
                </PageSection>
            </PageSection>
        </>
    );
}

export default ExceptionRequestDetailsPage;
