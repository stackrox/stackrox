import React from 'react';
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
    Text,
    TextVariants,
    Title,
} from '@patternfly/react-core';
import {
    ExpandableRowContent,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';
import { useParams, Link } from 'react-router-dom';
import pluralize from 'pluralize';

import { exceptionManagementPath } from 'routePaths';
import useSet from 'hooks/useSet';
import { VulnerabilityException } from 'services/VulnerabilityExceptionService';
import { getDateTime } from 'utils/dateUtils';
import { ensureExhaustive } from 'utils/type.utils';
import { vulnerabilityExceptions } from 'Containers/Vulnerabilities/ExceptionManagement/mockUtils';
import { getEntityPagePath } from 'Containers/Vulnerabilities/WorkloadCves/searchUtils';
import { VulnerabilitySeverityLabel } from 'Containers/Vulnerabilities/WorkloadCves/types';

import NotFoundMessage from 'Components/NotFoundMessage';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import {
    RequestExpires,
    RequestedAction,
    RequestCreatedAt,
    RequestScope,
} from './components/ExceptionRequestTableCells';
// @TODO: Move these files up to a common directory and move the types used in these files as well
import SeverityCountLabels from '../WorkloadCves/components/SeverityCountLabels';
import CvssTd from '../WorkloadCves/components/CvssTd';
import DateDistanceTd from '../WorkloadCves/components/DatePhraseTd';

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
    const expandedRowSet = useSet<string>();

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

    // @TODO: Will need to figure out a good way to distinguish the original request cves from the updated one
    const { cves } = vulnerabilityException;

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
            <PageSection>
                <PageSection variant="light">
                    <Flex direction={{ default: 'column' }}>
                        <Title headingLevel="h2">{cves.length} results found</Title>
                        <TableComposable variant="compact">
                            <Thead noWrap>
                                <Tr>
                                    <Td />
                                    <Th>CVE</Th>
                                    <Th>Images by severity</Th>
                                    <Th>CVSS</Th>
                                    <Th>Affected images</Th>
                                    <Th>First discovered</Th>
                                </Tr>
                            </Thead>
                            {cves.length === 0 && (
                                <Tbody>
                                    <Tr>
                                        <Td colSpan={6}>
                                            <Bullseye>
                                                <EmptyStateTemplate
                                                    title="No results found"
                                                    headingLevel="h2"
                                                    icon={SearchIcon}
                                                />
                                            </Bullseye>
                                        </Td>
                                    </Tr>
                                </Tbody>
                            )}
                            {cves.length !== 0 &&
                                cves.map((cve, rowIndex) => {
                                    const isExpanded = expandedRowSet.has(cve);

                                    // @TODO: Use real data later
                                    const criticalCount = 4;
                                    const importantCount = 3;
                                    const moderateCount = 4;
                                    const lowCount = 0;
                                    const filteredSeverities: VulnerabilitySeverityLabel[] = [
                                        'Critical',
                                        'Important',
                                        'Moderate',
                                        'Low',
                                    ];
                                    const scoreVersions = ['V3'];
                                    const affectedImageCount = 5;
                                    const firstDiscoveredInSystem = '2023-11-06T15:40:06.988717Z';

                                    return (
                                        <Tbody key={cve}>
                                            <Tr>
                                                <Td
                                                    expand={{
                                                        rowIndex,
                                                        isExpanded,
                                                        onToggle: () => expandedRowSet.toggle(cve),
                                                    }}
                                                />
                                                <Td>
                                                    <Link to={getEntityPagePath('CVE', cve)}>
                                                        {cve}
                                                    </Link>
                                                </Td>
                                                <Td>
                                                    <SeverityCountLabels
                                                        criticalCount={criticalCount}
                                                        importantCount={importantCount}
                                                        moderateCount={moderateCount}
                                                        lowCount={lowCount}
                                                        filteredSeverities={filteredSeverities}
                                                    />
                                                </Td>
                                                <Td>
                                                    <CvssTd
                                                        cvss={10}
                                                        scoreVersion={
                                                            scoreVersions.length > 0
                                                                ? scoreVersions.join('/')
                                                                : undefined
                                                        }
                                                    />
                                                </Td>
                                                <Td>{`${affectedImageCount} ${pluralize(
                                                    'image',
                                                    affectedImageCount
                                                )}`}</Td>
                                                <Td>
                                                    <DateDistanceTd
                                                        date={firstDiscoveredInSystem}
                                                    />
                                                </Td>
                                            </Tr>
                                            <Tr isExpanded={isExpanded}>
                                                <Td />
                                                <Td colSpan={5}>
                                                    <ExpandableRowContent>
                                                        <Text>
                                                            Contrary to popular belief, Lorem Ipsum
                                                            is not simply random text. It has roots
                                                            in a piece of classical Latin literature
                                                            from 45 BC, making it over 2000 years
                                                            old. Richard McClintock, a Latin
                                                            professor at Hampden-Sydney College in
                                                            Virginia, looked up one of the more
                                                            obscure Latin words, consectetur, from a
                                                            Lorem Ipsum passage, and going through
                                                            the cites of the word in classical
                                                            literature, discovered the undoubtable
                                                            source.
                                                        </Text>
                                                    </ExpandableRowContent>
                                                </Td>
                                            </Tr>
                                        </Tbody>
                                    );
                                })}
                        </TableComposable>
                    </Flex>
                </PageSection>
            </PageSection>
        </>
    );
}

export default ExceptionRequestDetailsPage;
