import React, { useState } from 'react';
import { Link, generatePath, useNavigate } from 'react-router-dom';
import isEmpty from 'lodash/isEmpty';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    PageSection,
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
    Text,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    SearchInput,
    Pagination,
    EmptyStateHeader,
} from '@patternfly/react-core';
import { DropdownItem } from '@patternfly/react-core/deprecated';
import { ActionsColumn, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { ExclamationCircleIcon, FileIcon, SearchIcon } from '@patternfly/react-icons';

import { vulnerabilityConfigurationReportsPath } from 'routePaths';
import { vulnerabilityConfigurationReportPath } from 'Containers/Vulnerabilities/VulnerablityReporting/pathsForVulnerabilityReporting';
import useFetchReports from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReports';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';
import useURLPagination from 'hooks/useURLPagination';
import useRunReport from 'Containers/Vulnerabilities/VulnerablityReporting/api/useRunReport';
import useDeleteModal, {
    isErrorDeleteResult,
    isSuccessDeleteResult,
} from 'Containers/Vulnerabilities/VulnerablityReporting/hooks/useDeleteModal';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { useWatchLastSnapshotForReports } from 'Containers/Vulnerabilities/VulnerablityReporting/api/useWatchLastSnapshotForReports';

import DeleteModal from 'Components/PatternFly/DeleteModal';
import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/EmptyStateTemplate/EmptyStateTemplate';
import CollectionsFormModal from 'Containers/Collections/CollectionFormModal';
import useToasts, { Toast } from 'hooks/patternfly/useToasts';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import useTableSelection from 'hooks/useTableSelection';
import pluralize from 'pluralize';
import HelpIconTh from 'Components/HelpIconTh';
import JobStatusPopoverContent from 'Components/ReportJob/JobStatusPopoverContent';
import MyLastJobStatus from 'Components/ReportJob/MyLastJobStatus';
import useAuthStatus from 'hooks/useAuthStatus';
import { reportDownloadURL } from 'services/ReportsService';

const CreateReportsButton = () => {
    return (
        <Link to={`${vulnerabilityConfigurationReportsPath}?action=create`}>
            <Button variant="primary">Create report</Button>
        </Link>
    );
};

const reportNameSearchKey = 'Report Name';

const sortOptions = {
    sortFields: [reportNameSearchKey],
    defaultSortOption: { field: reportNameSearchKey, direction: 'asc' } as const,
};

const emptyReportArray = [];

function ConfigReportsTab() {
    const navigate = useNavigate();
    const { currentUser } = useAuthStatus();

    const { hasReadWriteAccess, hasReadAccess } = usePermissions();
    const hasWriteAccessForReport =
        hasReadWriteAccess('WorkflowAdministration') &&
        hasReadAccess('Image') && // for vulnerabilities
        hasReadAccess('Integration'); // for notifiers

    const isRouteEnabled = useIsRouteEnabled();
    const isCollectionsRouteEnabled = isRouteEnabled('collections');

    const { toasts, addToast, removeToast } = useToasts();

    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [searchValue, setSearchValue] = useState(() => {
        return (searchFilter?.[reportNameSearchKey] as string) || '';
    });
    const [collectionModalId, setCollectionModalId] = useState<string | null>(null);

    const {
        reportConfigurations,
        totalReports,
        isLoading,
        error: fetchError,
        fetchReports,
    } = useFetchReports({
        searchFilter,
        page,
        perPage,
        sortOption,
    });
    const { reportSnapshots, isLoading: isLoadingReportSnapshots } =
        useWatchLastSnapshotForReports(reportConfigurations);
    const { isRunning, runError, runReport } = useRunReport({
        onCompleted: ({ reportNotificationMethod }) => {
            if (reportNotificationMethod === 'EMAIL') {
                addToast('The report has been sent to the configured email notifier', 'success');
            } else if (reportNotificationMethod === 'DOWNLOAD') {
                addToast(
                    'The report generation has started and will be available for download once complete',
                    'success'
                );
            }
            fetchReports();
        },
    });

    const {
        selected,
        numSelected,
        allRowsSelected,
        hasSelections,
        onSelect,
        onSelectAll,
        onClearAll: onClearAllSelected,
        getSelectedIds,
    } = useTableSelection(reportConfigurations || emptyReportArray);

    const {
        openDeleteModal,
        isDeleteModalOpen,
        closeDeleteModal,
        isDeleting,
        onDelete,
        deleteResults,
        reportIdsToDelete,
    } = useDeleteModal({
        onCompleted: () => {
            onClearAllSelected();
            fetchReports();
        },
    });

    function onConfirmDeleteSelection() {
        const selectedIds = getSelectedIds();
        openDeleteModal(selectedIds);
    }

    const numSuccessfulDeletions = deleteResults?.filter(isSuccessDeleteResult).length || 0;

    return (
        <>
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }: Toast) => (
                    <Alert
                        key={key}
                        variant={variant}
                        title={title}
                        component="p"
                        timeout
                        onTimeout={() => removeToast(key)}
                        actionClose={
                            <AlertActionCloseButton
                                title={title}
                                variantLabel={variant}
                                onClose={() => removeToast(key)}
                            />
                        }
                    >
                        {children}
                    </Alert>
                ))}
            </AlertGroup>
            <PageTitle title="Vulnerability reporting" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                {runError && <Alert variant="danger" isInline title={runError} component="p" />}
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    className="pf-v5-u-py-lg pf-v5-u-px-lg"
                >
                    <Text>
                        Configure reports, define collections, and assign delivery destinations to
                        report on vulnerabilities across the organization.
                    </Text>
                    {reportConfigurations &&
                        reportConfigurations.length > 0 &&
                        hasWriteAccessForReport && (
                            <FlexItem align={{ default: 'alignRight' }}>
                                <CreateReportsButton />
                            </FlexItem>
                        )}
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody className="pf-v5-u-p-0">
                            <Toolbar>
                                <ToolbarContent>
                                    <ToolbarItem
                                        variant="search-filter"
                                        className="pf-v5-u-flex-grow-1"
                                    >
                                        <SearchInput
                                            placeholder="Filter by report name"
                                            value={searchValue}
                                            onChange={(_event, value) => setSearchValue(value)}
                                            onSearch={(_event, value) => {
                                                setSearchValue(value);
                                                setSearchFilter({ [reportNameSearchKey]: value });
                                                setPage(1);
                                            }}
                                            onClear={() => {
                                                setSearchValue('');
                                                setSearchFilter({});
                                                setPage(1);
                                            }}
                                        />
                                    </ToolbarItem>
                                    <ToolbarItem>
                                        <BulkActionsDropdown isDisabled={!hasSelections}>
                                            <DropdownItem
                                                key="delete"
                                                component="button"
                                                onClick={onConfirmDeleteSelection}
                                            >
                                                Delete ({numSelected})
                                            </DropdownItem>
                                        </BulkActionsDropdown>
                                    </ToolbarItem>
                                    <ToolbarItem
                                        variant="pagination"
                                        align={{ default: 'alignRight' }}
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
                            {isLoading && !reportConfigurations && (
                                <div className="pf-v5-u-p-md">
                                    <Bullseye>
                                        <Spinner />
                                    </Bullseye>
                                </div>
                            )}
                            {fetchError && (
                                <EmptyState variant="sm">
                                    <EmptyStateHeader
                                        titleText="Unable to get vulnerability reports"
                                        icon={
                                            <EmptyStateIcon
                                                icon={ExclamationCircleIcon}
                                                className="pf-v5-u-danger-color-100"
                                            />
                                        }
                                        headingLevel="h2"
                                    />
                                    <EmptyStateBody>{fetchError}</EmptyStateBody>
                                </EmptyState>
                            )}
                            {reportConfigurations && (
                                <Table borders={false}>
                                    <Thead noWrap>
                                        <Tr>
                                            <Th
                                                select={{
                                                    onSelect: onSelectAll,
                                                    isSelected: allRowsSelected,
                                                }}
                                            />
                                            <Th sort={getSortParams(reportNameSearchKey)}>
                                                Report
                                            </Th>
                                            <HelpIconTh
                                                popoverContent={
                                                    <div>
                                                        A set of user-configured rules for selecting
                                                        deployments as part of the collection
                                                    </div>
                                                }
                                            >
                                                Collection
                                            </HelpIconTh>
                                            <Th>Description</Th>
                                            <HelpIconTh
                                                popoverContent={<JobStatusPopoverContent />}
                                            >
                                                My last job status
                                            </HelpIconTh>
                                            {hasWriteAccessForReport && (
                                                <Th>
                                                    <span className="pf-v5-screen-reader">
                                                        Row actions
                                                    </span>
                                                </Th>
                                            )}
                                        </Tr>
                                    </Thead>
                                    {reportConfigurations.length === 0 && isEmpty(searchFilter) && (
                                        <Tbody>
                                            <Tr>
                                                <Td colSpan={6}>
                                                    <Bullseye>
                                                        <EmptyStateTemplate
                                                            title="No vulnerability reports yet"
                                                            headingLevel="h2"
                                                            icon={FileIcon}
                                                        >
                                                            {hasWriteAccessForReport && (
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
                                    {reportConfigurations.length === 0 &&
                                        !isEmpty(searchFilter) && (
                                            <Tbody>
                                                <Tr>
                                                    <Td colSpan={6}>
                                                        <Bullseye>
                                                            <EmptyStateTemplate
                                                                title="No results found"
                                                                headingLevel="h2"
                                                                icon={SearchIcon}
                                                            >
                                                                {hasWriteAccessForReport && (
                                                                    <Flex
                                                                        direction={{
                                                                            default: 'column',
                                                                        }}
                                                                    >
                                                                        <FlexItem>
                                                                            <Text>
                                                                                No results match
                                                                                this filter
                                                                                criteria. Clear the
                                                                                filter and try
                                                                                again.
                                                                            </Text>
                                                                        </FlexItem>
                                                                        <FlexItem>
                                                                            <Button
                                                                                variant="link"
                                                                                onClick={() => {
                                                                                    setSearchValue(
                                                                                        ''
                                                                                    );
                                                                                    setSearchFilter(
                                                                                        {}
                                                                                    );
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
                                    {reportConfigurations.map((report, rowIndex) => {
                                        const vulnReportURL = generatePath(
                                            vulnerabilityConfigurationReportPath,
                                            {
                                                reportId: report.id,
                                            }
                                        );
                                        const snapshot = reportSnapshots[report.id];
                                        const isReportStatusPending =
                                            snapshot?.reportStatus.runState === 'PREPARING' ||
                                            snapshot?.reportStatus.runState === 'WAITING';
                                        const rowActions = [
                                            {
                                                title: 'Edit report',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    navigate(`${vulnReportURL}?action=edit`);
                                                },
                                                isDisabled: isReportStatusPending,
                                            },
                                            {
                                                isSeparator: true,
                                            },
                                            {
                                                title: 'Send report',
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
                                                    navigate(`${vulnReportURL}?action=clone`);
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
                                                                ? 'pf-v5-u-danger-color-100'
                                                                : ''
                                                        }
                                                    >
                                                        Delete report
                                                    </span>
                                                ),
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    openDeleteModal([report.id]);
                                                },
                                                isDisabled: isReportStatusPending,
                                            },
                                        ];
                                        const { collectionName, collectionId } =
                                            report.resourceScope.collectionScope;

                                        return (
                                            <Tbody
                                                key={report.id}
                                                style={{
                                                    borderBottom:
                                                        '1px solid var(--pf-v5-c-table--BorderColor)',
                                                }}
                                            >
                                                <Tr>
                                                    <Td
                                                        key={report.id}
                                                        select={{
                                                            rowIndex,
                                                            onSelect,
                                                            isSelected: selected[rowIndex],
                                                        }}
                                                    />
                                                    <Td dataLabel="Report">
                                                        <Link to={vulnReportURL}>
                                                            {report.name}
                                                        </Link>
                                                    </Td>
                                                    <Td dataLabel="Collection">
                                                        {isCollectionsRouteEnabled ? (
                                                            <Button
                                                                variant="link"
                                                                isInline
                                                                onClick={() =>
                                                                    setCollectionModalId(
                                                                        collectionId
                                                                    )
                                                                }
                                                            >
                                                                {collectionName}
                                                            </Button>
                                                        ) : (
                                                            collectionName
                                                        )}
                                                    </Td>
                                                    <Td dataLabel="Description">
                                                        {report.description || '-'}
                                                    </Td>
                                                    <Td dataLabel="My last job status">
                                                        <MyLastJobStatus
                                                            snapshot={snapshot}
                                                            isLoadingSnapshots={
                                                                isLoadingReportSnapshots
                                                            }
                                                            currentUserId={currentUser.userId}
                                                            baseDownloadURL={reportDownloadURL}
                                                        />
                                                    </Td>
                                                    {hasWriteAccessForReport && (
                                                        <Td isActionCell>
                                                            <ActionsColumn
                                                                items={rowActions}
                                                                isDisabled={isRunning}
                                                                // menuAppendTo={() => document.body}
                                                            />
                                                        </Td>
                                                    )}
                                                </Tr>
                                            </Tbody>
                                        );
                                    })}
                                </Table>
                            )}
                        </CardBody>
                    </Card>
                </PageSection>
            </PageSection>
            <DeleteModal
                title={`Permanently delete (${reportIdsToDelete.length}) ${pluralize(
                    'report',
                    reportIdsToDelete.length
                )}?`}
                isOpen={isDeleteModalOpen}
                onClose={closeDeleteModal}
                isDeleting={isDeleting}
                onDelete={onDelete}
            >
                <AlertGroup>
                    {numSuccessfulDeletions > 0 && (
                        <Alert
                            isInline
                            variant="success"
                            title={`Successfully deleted ${numSuccessfulDeletions} ${pluralize(
                                'report',
                                numSuccessfulDeletions
                            )}`}
                            component="p"
                            className="pf-v5-u-mb-sm"
                        />
                    )}
                    {deleteResults?.filter(isErrorDeleteResult).map((deleteResult) => {
                        const report = reportConfigurations?.find(
                            (reportConfig) => reportConfig.id === deleteResult.id
                        );
                        if (!report) {
                            return null;
                        }
                        return (
                            <Alert
                                isInline
                                variant="danger"
                                title={`Failed to delete "${report.name}"`}
                                component="p"
                                className="pf-v5-u-mb-sm"
                            >
                                {deleteResult.error}
                            </Alert>
                        );
                    })}
                </AlertGroup>
                <p>
                    The selected report(s) and any attached downloadable reports will be permanently
                    deleted. The action cannot be undone.
                </p>
            </DeleteModal>
            {collectionModalId && (
                <CollectionsFormModal
                    hasWriteAccessForCollections={false}
                    modalAction={{ type: 'view', collectionId: collectionModalId }}
                    onClose={() => setCollectionModalId(null)}
                />
            )}
        </>
    );
}

export default ConfigReportsTab;
