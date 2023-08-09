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
import isEmpty from 'lodash/isEmpty';

import { vulnerabilityReportsPath } from 'routePaths';
import { vulnerabilityReportPath } from 'Containers/Vulnerabilities/VulnerablityReporting/pathsForVulnerabilityReporting';
import useFetchReports from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReports';
import usePermissions from 'hooks/usePermissions';
import useURLPagination from 'hooks/useURLPagination';
import useRunReport from 'Containers/Vulnerabilities/VulnerablityReporting/api/useRunReport';
import useDeleteModal from 'Containers/Vulnerabilities/VulnerablityReporting/hooks/useDeleteModal';
import useURLSearch from 'hooks/useURLSearch';

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import HelpIconTh from './HelpIconTh';
import MyActiveJobStatus from './MyActiveJobStatus';
import DeleteReportModal from '../components/DeleteReportModal';

const CreateReportsButton = () => {
    return (
        <Link to={`${vulnerabilityReportsPath}?action=create`}>
            <Button variant="primary">Create report</Button>
        </Link>
    );
};

const reportNameSearchKey = 'Report Name';

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

    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [searchValue, setSearchValue] = useState(() => {
        return (searchFilter?.[reportNameSearchKey] as string) || '';
    });

    const {
        reports,
        totalReports,
        isLoading,
        error: fetchError,
        fetchReports,
    } = useFetchReports({
        searchFilter,
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
                                                setSearchFilter({ [reportNameSearchKey]: value });
                                            }}
                                            onClear={() => {
                                                setSearchValue('');
                                                setSearchFilter({});
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
                                            onSetPage={(_, newPage) => setPage(newPage)}
                                            onPerPageSelect={(_, newPerPage) =>
                                                setPerPage(newPerPage)
                                            }
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
                                            <HelpIconTh
                                                popoverContent={
                                                    <div>
                                                        A set of user-configured rules for selecting
                                                        deployments as part of the report scope
                                                    </div>
                                                }
                                            >
                                                Collection
                                            </HelpIconTh>
                                            <Th>Description</Th>
                                            <HelpIconTh
                                                popoverContent={
                                                    <Flex
                                                        direction={{ default: 'column' }}
                                                        spaceItems={{ default: 'spaceItemsMd' }}
                                                    >
                                                        <FlexItem>
                                                            <p>
                                                                The status of your last requested
                                                                job from the active job queue. An
                                                                active job queue includes any
                                                                requested job with the status of
                                                                preparing or waiting until completed
                                                            </p>
                                                        </FlexItem>
                                                        <FlexItem>
                                                            <p>Preparing:</p>
                                                            <p>
                                                                Your last requested job is still
                                                                being processed
                                                            </p>
                                                        </FlexItem>
                                                        <FlexItem>
                                                            <p>Waiting:</p>
                                                            <p>
                                                                Your last requested job is in the
                                                                queue and waiting to be processed
                                                                since other users requested their
                                                                jobs before you.
                                                            </p>
                                                        </FlexItem>
                                                    </Flex>
                                                }
                                            >
                                                My active job status
                                            </HelpIconTh>
                                            <Td />
                                        </Tr>
                                    </Thead>
                                    {reports.length === 0 && isEmpty(searchFilter) && (
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
                                    {reports.length === 0 && !isEmpty(searchFilter) && (
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
                                                                                setSearchFilter({});
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
                                        const isReportStatusPending =
                                            report.reportSnapshot?.reportStatus.runState ===
                                                'PREPARING' ||
                                            report.reportSnapshot?.reportStatus.runState ===
                                                'WAITING';
                                        const rowActions = [
                                            {
                                                title: 'Edit report',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    history.push(`${vulnReportURL}?action=edit`);
                                                },
                                                isDisabled: isReportStatusPending,
                                            },
                                            {
                                                isSeparator: true,
                                            },
                                            {
                                                title: 'Send report now',
                                                description:
                                                    report.notifiers.length === 0
                                                        ? 'No delivery destinations set'
                                                        : '',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    runReport(report.id, 'EMAIL');
                                                },
                                                isDisabled:
                                                    isReportStatusPending ||
                                                    report.notifiers.length === 0,
                                            },
                                            {
                                                title: 'Generate download',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    runReport(report.id, 'DOWNLOAD');
                                                },
                                                isDisabled: isReportStatusPending,
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
                                                    <span
                                                        className={
                                                            !isReportStatusPending
                                                                ? 'pf-u-danger-color-100'
                                                                : ''
                                                        }
                                                    >
                                                        Delete report
                                                    </span>
                                                ),
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    openDeleteModal(report.id);
                                                },
                                                isDisabled: isReportStatusPending,
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
                                                    <Td>{report.description || '-'}</Td>
                                                    <Td>
                                                        <MyActiveJobStatus
                                                            reportStatus={
                                                                report.reportSnapshot?.reportStatus
                                                            }
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
