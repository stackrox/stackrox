import React, { useState } from 'react';
import { Link, generatePath, useHistory } from 'react-router-dom';
import isEmpty from 'lodash/isEmpty';
import {
    AlertActionCloseButton,
    AlertGroup,
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
    DropdownItem,
} from '@patternfly/react-core';
import { ActionsColumn, TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { ExclamationCircleIcon, FileIcon, SearchIcon } from '@patternfly/react-icons';

import { vulnerabilityReportsPath } from 'routePaths';
import { vulnerabilityReportPath } from 'Containers/Vulnerabilities/VulnerablityReporting/pathsForVulnerabilityReporting';
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

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import CollectionsFormModal from 'Containers/Collections/CollectionFormModal';
import useToasts, { Toast } from 'hooks/patternfly/useToasts';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import useTableSelection from 'hooks/useTableSelection';
import pluralize from 'pluralize';
import HelpIconTh from './HelpIconTh';
import MyActiveJobStatus from './MyActiveJobStatus';
import DeleteModal from '../components/DeleteModal';

const CreateReportsButton = () => {
    return (
        <Link to={`${vulnerabilityReportsPath}?action=create`}>
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

function VulnReportsPage() {
    const history = useHistory();

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
    const { reportSnapshots } = useWatchLastSnapshotForReports(reportConfigurations);
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
                                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                    <Title headingLevel="h1">Vulnerability reporting</Title>
                                </Flex>
                            </FlexItem>
                            <FlexItem>
                                Configure reports, define report scopes, and assign delivery
                                destinations to report on vulnerabilities across the organization.
                            </FlexItem>
                        </Flex>
                    </FlexItem>
                    {reportConfigurations &&
                        reportConfigurations.length > 0 &&
                        hasWriteAccessForReport && (
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
                            {isLoading && !reportConfigurations && (
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
                            {reportConfigurations && (
                                <TableComposable borders={false}>
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
                                                                job from the{' '}
                                                                <strong>active job queue</strong>.
                                                                An <strong>active job queue</strong>{' '}
                                                                includes any requested job with the
                                                                status of <strong>preparing</strong>{' '}
                                                                or <strong>waiting</strong> until
                                                                completed.
                                                            </p>
                                                        </FlexItem>
                                                        <FlexItem>
                                                            <p>
                                                                <strong>Preparing:</strong>
                                                            </p>
                                                            <p>
                                                                Your last requested job is still
                                                                being processed.
                                                            </p>
                                                        </FlexItem>
                                                        <FlexItem>
                                                            <p>
                                                                <strong>Waiting:</strong>
                                                            </p>
                                                            <p>
                                                                Your last requested job is in the
                                                                queue and waiting to be processed
                                                                since other requested jobs are being
                                                                processed.
                                                            </p>
                                                        </FlexItem>
                                                    </Flex>
                                                }
                                            >
                                                My active job status
                                            </HelpIconTh>
                                            {hasWriteAccessForReport && <Td />}
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
                                            vulnerabilityReportPath,
                                            {
                                                reportId: report.id,
                                            }
                                        ) as string;
                                        const reportSnapshot = reportSnapshots[report.id];
                                        const isReportStatusPending =
                                            reportSnapshot?.reportStatus.runState === 'PREPARING' ||
                                            reportSnapshot?.reportStatus.runState === 'WAITING';
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
                                                        '1px solid var(--pf-c-table--BorderColor)',
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
                                                    <Td>
                                                        <Link to={vulnReportURL}>
                                                            {report.name}
                                                        </Link>
                                                    </Td>
                                                    <Td>
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
                                                    <Td>{report.description || '-'}</Td>
                                                    <Td>
                                                        <MyActiveJobStatus
                                                            reportStatus={
                                                                reportSnapshot?.reportStatus
                                                            }
                                                        />
                                                    </Td>
                                                    {hasWriteAccessForReport && (
                                                        <Td isActionCell>
                                                            <ActionsColumn
                                                                items={rowActions}
                                                                isDisabled={isRunning}
                                                                menuAppendTo={() => document.body}
                                                            />
                                                        </Td>
                                                    )}
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
                            variant={AlertVariant.success}
                            title={`Successfully deleted ${numSuccessfulDeletions} ${pluralize(
                                'report',
                                numSuccessfulDeletions
                            )}`}
                            className="pf-u-mb-sm"
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
                                variant={AlertVariant.danger}
                                title={`Failed to delete "${report.name}"`}
                                className="pf-u-mb-sm"
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

export default VulnReportsPage;
