import React from 'react';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Button,
    Card,
    CardBody,
    Bullseye,
    Spinner,
    EmptyState,
    EmptyStateIcon,
    EmptyStateBody,
    EmptyStateVariant,
    Text,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Link } from 'react-router-dom';
import { ExclamationCircleIcon, FileIcon } from '@patternfly/react-icons';

import useFetchReports from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReports';
import usePermissions from 'hooks/usePermissions';
import { vulnerabilityReportsPath } from 'routePaths';

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import HelpIconTh from './HelpIconTh';
import LastRunStatusState from './LastRunStatusState';
import LastRunState from './LastRunState';

const CreateReportsButton = () => {
    return (
        <Link to={`${vulnerabilityReportsPath}?action=create`}>
            <Button variant="primary">Create report</Button>
        </Link>
    );
};

function VulnReportsPage() {
    const { hasReadWriteAccess, hasReadAccess } = usePermissions();

    const hasWorkflowAdministrationWriteAccess = hasReadWriteAccess('WorkflowAdministration');
    const hasImageReadAccess = hasReadAccess('Image');
    const hasAccessScopeReadAccess = hasReadAccess('Access');
    const hasNotifierIntegrationReadAccess = hasReadAccess('Integration');
    const canCreateReports =
        hasWorkflowAdministrationWriteAccess &&
        hasImageReadAccess &&
        hasAccessScopeReadAccess &&
        hasNotifierIntegrationReadAccess;

    const { reports, isLoading, error } = useFetchReports();

    return (
        <>
            <PageTitle title="Vulnerability reporting" />
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    className="pf-u-py-lg pf-u-px-lg"
                >
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Flex direction={{ default: 'column' }}>
                            <FlexItem>
                                <Title headingLevel="h1">Vulnerability reporting</Title>
                            </FlexItem>
                            <FlexItem>
                                Configure reports, define report scopes, and assign delivery
                                destinations to report on vulnerabilities across the organization.
                            </FlexItem>
                        </Flex>
                    </FlexItem>
                    {reports.length > 0 && canCreateReports && (
                        <FlexItem>
                            <CreateReportsButton />
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody className="pf-u-p-0">
                            {isLoading && (
                                <div className="pf-u-p-md">
                                    <Bullseye>
                                        <Spinner isSVG />
                                    </Bullseye>
                                </div>
                            )}
                            {error && (
                                <EmptyState variant={EmptyStateVariant.small}>
                                    <EmptyStateIcon
                                        icon={ExclamationCircleIcon}
                                        className="pf-u-danger-color-100"
                                    />
                                    <Title headingLevel="h2" size="lg">
                                        Unable to get vulnerability reports
                                    </Title>
                                    <EmptyStateBody>{error}</EmptyStateBody>
                                </EmptyState>
                            )}
                            {!isLoading && !error && (
                                <TableComposable borders={false}>
                                    <Thead noWrap>
                                        <Tr>
                                            <Th>Report</Th>
                                            <HelpIconTh tooltip="A set of user-configured rules for selecting deployments as part of the report scope">
                                                Collection
                                            </HelpIconTh>
                                            <Th>Last run status</Th>
                                            <HelpIconTh tooltip="The report that was last run by a schedule or an on-demand action including 'send report now' and 'generate a downloadable report'">
                                                Last run
                                            </HelpIconTh>
                                        </Tr>
                                    </Thead>
                                    {reports.length === 0 && (
                                        <Tbody>
                                            <Tr>
                                                <Td colSpan={4}>
                                                    <Bullseye>
                                                        <EmptyStateTemplate
                                                            title="No vulnerability reports yet"
                                                            headingLevel="h2"
                                                            icon={FileIcon}
                                                        >
                                                            {canCreateReports && (
                                                                <Flex
                                                                    direction={{
                                                                        default: 'column',
                                                                    }}
                                                                >
                                                                    <FlexItem>
                                                                        <Text>
                                                                            To get started, create a
                                                                            report
                                                                        </Text>
                                                                    </FlexItem>
                                                                    <FlexItem>
                                                                        <CreateReportsButton />
                                                                    </FlexItem>
                                                                </Flex>
                                                            )}
                                                        </EmptyStateTemplate>
                                                    </Bullseye>
                                                </Td>
                                            </Tr>
                                        </Tbody>
                                    )}
                                    {reports.map((report) => {
                                        return (
                                            <Tbody
                                                key={report.id}
                                                style={{
                                                    borderBottom:
                                                        '1px solid var(--pf-c-table--BorderColor)',
                                                }}
                                            >
                                                <Tr>
                                                    <Td>{report.name}</Td>
                                                    <Td>
                                                        {
                                                            report.resourceScope.collectionScope
                                                                .collectionName
                                                        }
                                                    </Td>
                                                    <Td>
                                                        <LastRunStatusState
                                                            reportStatus={
                                                                report.reportLastRunStatus
                                                            }
                                                        />
                                                    </Td>
                                                    <Td>
                                                        <LastRunState
                                                            reportStatus={report.reportStatus}
                                                        />
                                                    </Td>
                                                </Tr>
                                            </Tbody>
                                        );
                                    })}
                                </TableComposable>
                            )}
                        </CardBody>
                    </Card>
                </PageSection>
            </PageSection>
        </>
    );
}

export default VulnReportsPage;
