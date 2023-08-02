import React, { useState } from 'react';
import {
    PageSection,
    Title,
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
    Alert,
    AlertVariant,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    SearchInput,
    Pagination,
} from '@patternfly/react-core';
import { ActionsColumn, TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Link, generatePath, useHistory } from 'react-router-dom';
import { ExclamationCircleIcon, FileIcon, SearchIcon } from '@patternfly/react-icons';

import useFetchReports from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReports';
import usePermissions from 'hooks/usePermissions';
import { vulnerabilityReportsPath } from 'routePaths';

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';

import useRunReport from 'Containers/Vulnerabilities/VulnerablityReporting/api/useRunReport';
import usePagination from 'hooks/patternfly/usePagination';
import { vulnerabilityReportPath } from '../pathsForVulnerabilityReporting';
import HelpIconTh from './HelpIconTh';
import LastRunStatusState from './LastRunStatusState';
import LastRunState from './LastRunState';
import useDeleteModal from '../hooks/useDeleteModal';
import DeleteReportModal from '../components/DeleteReportModal';

const CreateReportsButton = () => {
    return (
        <Link to={`${vulnerabilityReportsPath}?action=create`}>
            <Button variant="primary">Create report</Button>
        </Link>
    );
};

function VulnReportsPage() {
    const history = useHistory();

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

    const [searchValue, setSearchValue] = useState('');
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const [query, setQuery] = useState('');

    const {
        reports,
        totalReports,
        isLoading,
        error: fetchError,
        fetchReports,
    } = useFetchReports({
        query,
        page,
        perPage,
    });
    const { isRunning, runError, runReport } = useRunReport({
        onCompleted: fetchReports,
    });

    const {
        openDeleteModal,
        isDeleteModalOpen,
        closeDeleteModal,
        isDeleting,
        onDelete,
        deleteError,
    } = useDeleteModal({
        onCompleted: fetchReports,
    });

    return (
        <>
            <PageTitle title="Vulnerability reporting" />
            {runError && <Alert variant={AlertVariant.danger} isInline title={runError} />}
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
                            <Toolbar>
                                <ToolbarContent>
                                    <ToolbarItem
                                        variant="search-filter"
                                        className="pf-u-flex-grow-1"
                                    >
                                        <SearchInput
                                            placeholder="Filter by report name"
                                            value={searchValue}
                                            onChange={(_event, value) => setSearchValue(value)}
                                            onSearch={(_event, value) => {
                                                setSearchValue(value);
                                                setQuery(`Report Name:${value}`);
                                            }}
                                            onClear={() => {
                                                setSearchValue('');
                                                setQuery('');
                                            }}
                                        />
                                    </ToolbarItem>
                                    <ToolbarItem
                                        variant="pagination"
                                        alignment={{ default: 'alignRight' }}
                                    >
                                        <Pagination
                                            itemCount={totalReports}
                                            page={page}
                                            perPage={perPage}
                                            onSetPage={onSetPage}
                                            onPerPageSelect={onPerPageSelect}
                                            isCompact
                                        />
                                    </ToolbarItem>
                                </ToolbarContent>
                            </Toolbar>
                            {isLoading && (
                                <div className="pf-u-p-md">
                                    <Bullseye>
                                        <Spinner isSVG />
                                    </Bullseye>
                                </div>
                            )}
                            {fetchError && (
                                <EmptyState variant={EmptyStateVariant.small}>
                                    <EmptyStateIcon
                                        icon={ExclamationCircleIcon}
                                        className="pf-u-danger-color-100"
                                    />
                                    <Title headingLevel="h2" size="lg">
                                        Unable to get vulnerability reports
                                    </Title>
                                    <EmptyStateBody>{fetchError}</EmptyStateBody>
                                </EmptyState>
                            )}
                            {!isLoading && !fetchError && (
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
                                            <Td />
                                        </Tr>
                                    </Thead>
                                    {reports.length === 0 && query === '' && (
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
                                    {reports.length === 0 && query !== '' && (
                                        <Tbody>
                                            <Tr>
                                                <Td colSpan={8}>
                                                    <Bullseye>
                                                        <EmptyStateTemplate
                                                            title="No results found"
                                                            headingLevel="h2"
                                                            icon={SearchIcon}
                                                        >
                                                            {canCreateReports && (
                                                                <Flex
                                                                    direction={{
                                                                        default: 'column',
                                                                    }}
                                                                >
                                                                    <FlexItem>
                                                                        <Text>
                                                                            No results match this
                                                                            filter criteria. Clear
                                                                            the filter and try
                                                                            again.
                                                                        </Text>
                                                                    </FlexItem>
                                                                    <FlexItem>
                                                                        <Button
                                                                            variant="link"
                                                                            onClick={() => {
                                                                                setSearchValue('');
                                                                                setQuery('');
                                                                            }}
                                                                        >
                                                                            Clear filter
                                                                        </Button>
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
                                        const vulnReportURL = generatePath(
                                            vulnerabilityReportPath,
                                            {
                                                reportId: report.id,
                                            }
                                        ) as string;
                                        const rowActions = [
                                            {
                                                title: 'Edit report',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    history.push(`${vulnReportURL}?action=edit`);
                                                },
                                            },
                                            {
                                                isSeparator: true,
                                            },
                                            {
                                                title: 'Send report now',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    runReport(report.id, 'EMAIL');
                                                },
                                            },
                                            {
                                                title: 'Generate download',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    runReport(report.id, 'DOWNLOAD');
                                                },
                                            },
                                            {
                                                title: 'Clone report',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    history.push(`${vulnReportURL}?action=clone`);
                                                },
                                            },
                                            {
                                                isSeparator: true,
                                            },
                                            {
                                                title: (
                                                    <span className="pf-u-danger-color-100">
                                                        Delete report
                                                    </span>
                                                ),
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    openDeleteModal(report.id);
                                                },
                                            },
                                        ];
                                        return (
                                            <Tbody
                                                key={report.id}
                                                style={{
                                                    borderBottom:
                                                        '1px solid var(--pf-c-table--BorderColor)',
                                                }}
                                            >
                                                <Tr>
                                                    <Td>
                                                        <Link to={vulnReportURL}>
                                                            {report.name}
                                                        </Link>
                                                    </Td>
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
                                                    <Td isActionCell>
                                                        <ActionsColumn
                                                            items={rowActions}
                                                            isDisabled={isRunning}
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
            <DeleteReportModal
                isOpen={isDeleteModalOpen}
                onClose={closeDeleteModal}
                isDeleting={isDeleting}
                onDelete={onDelete}
                error={deleteError}
            />
        </>
    );
}

export default VulnReportsPage;
