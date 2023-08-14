import React, { useState, ReactElement } from 'react';
import {
    Alert,
    AlertGroup,
    AlertVariant,
    Bullseye,
    Button,
    ButtonVariant,
    DropdownItem,
    Divider,
    PageSection,
    PageSectionVariants,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import pluralize from 'pluralize';

import usePermissions from 'hooks/usePermissions';
import useTableSelection from 'hooks/useTableSelection';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import LinkShim from 'Components/PatternFly/LinkShim';
import SearchFilterResults from 'Components/PatternFly/SearchFilterResults';
import TableCell from 'Components/PatternFly/TableCell';
import { vulnManagementReportsPath } from 'routePaths';
import { ReportConfiguration } from 'types/report.proto';
import { SearchFilter } from 'types/search';
import { TableColumn, SortDirection } from 'types/table';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { getWriteAccessForReport } from './VulnMgmtReport.utils';
import ReportsSearchFilter from './Components/ReportsSearchFilter';

export type ActionItem = {
    title: string | ReactElement;
    onClick: (item) => void;
};

type AlertInfo = {
    title: string;
    variant: AlertVariant;
    key: number;
    timeout?: number | boolean;
};

type ReportingTablePanelProps = {
    reports: ReportConfiguration[];
    reportCount: number;
    currentPage: number;
    setCurrentPage: (page) => void;
    perPage: number;
    setPerPage: (perPage) => void;
    searchFilter: SearchFilter;
    setSearchFilter: (SearchFilter) => void;
    activeSortIndex: number;
    setActiveSortIndex: (idx) => void;
    activeSortDirection: SortDirection;
    setActiveSortDirection: (dir) => void;
    columns: TableColumn[];
    onRunReports: (reportIds: string[]) => Promise<void[]>; // return value not used
    onDeleteReports: (reportIds: string[]) => Promise<void>; // return value not used
};

function ReportingTablePanel({
    reports,
    reportCount,
    currentPage,
    setCurrentPage,
    perPage,
    setPerPage,
    searchFilter,
    setSearchFilter,
    activeSortIndex,
    setActiveSortIndex,
    activeSortDirection,
    setActiveSortDirection,
    columns,
    onRunReports,
    onDeleteReports,
}: ReportingTablePanelProps): ReactElement {
    const [alerts, setAlerts] = useState<AlertInfo[]>([]);
    const [deletingReportIds, setDeletingReportIds] = useState<string[]>([]);

    const { hasReadWriteAccess, hasReadAccess } = usePermissions();
    const hasWriteAccessForReport = getWriteAccessForReport({ hasReadAccess, hasReadWriteAccess });

    const {
        selected,
        numSelected,
        allRowsSelected,
        hasSelections,
        onSelect,
        onSelectAll,
        onClearAll,
        getSelectedIds,
    } = useTableSelection(reports);

    function onDeleteSelected() {
        const idsToDelete = getSelectedIds();
        setDeletingReportIds(idsToDelete);
    }

    // Handle page changes.
    function changePage(_, newPage) {
        if (newPage !== currentPage) {
            setCurrentPage(newPage);
        }
    }

    function changePerPage(_, newPerPage) {
        setPerPage(newPerPage);
    }

    function onSort(e, index, direction) {
        setActiveSortIndex(index);
        setActiveSortDirection(direction);
    }

    function onClickDelete(ids) {
        setDeletingReportIds(ids);
    }

    function onClickRun(ids) {
        setAlerts([]);

        onRunReports(ids)
            .then(() => {
                const message = 'The report has been queued to run.';
                const alertInfo = {
                    title: message,
                    variant: AlertVariant.success,
                    key: new Date().getTime(),
                    timeout: 6000,
                };
                setAlerts((prevAlertInfo) => [...prevAlertInfo, alertInfo]);
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const alertInfo = {
                    title:
                        `Could not run report: ${message}` ||
                        'An unknown error occurred while triggering a report run',
                    variant: AlertVariant.danger,
                    key: new Date().getTime(),
                };
                setAlerts((prevAlertInfo) => [...prevAlertInfo, alertInfo]);
            });
    }

    function onConfirmDeletingReportIds() {
        setAlerts([]);

        onDeleteReports(deletingReportIds)
            .then(() => {
                setDeletingReportIds([]);
                onClearAll();
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const alertInfo = {
                    title:
                        `Could not delete report: ${message}` ||
                        'An unknown error occurred while deleting',
                    variant: AlertVariant.danger,
                    key: new Date().getTime(),
                };
                setAlerts((prevAlertInfo) => [...prevAlertInfo, alertInfo]);

                setDeletingReportIds([]);
            });
    }

    function onCancelDeleteReportIds() {
        setDeletingReportIds([]);
    }

    const deleteConfirmationText = `Are you sure you want to delete ${
        deletingReportIds.length
    } ${pluralize('report', deletingReportIds.length)}?`;

    return (
        <>
            {alerts.length > 0 && (
                <PageSection padding={{ default: 'padding' }} variant={PageSectionVariants.light}>
                    <AlertGroup
                        isLiveRegion
                        aria-live="polite"
                        aria-relevant="additions text"
                        aria-atomic="false"
                    >
                        {alerts.map(({ title, variant, key, timeout }) => (
                            <Alert
                                isInline
                                variant={variant}
                                title={title}
                                key={key}
                                timeout={timeout}
                                onTimeout={() => {
                                    setAlerts((prevAlerts) => {
                                        return prevAlerts.filter((alert) => alert.key !== key);
                                    });
                                }}
                            />
                        ))}
                    </AlertGroup>
                </PageSection>
            )}
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem>
                        <ReportsSearchFilter
                            searchFilter={searchFilter}
                            setSearchFilter={setSearchFilter}
                        />
                    </ToolbarItem>
                    <ToolbarItem variant="separator" />
                    <ToolbarItem>
                        <BulkActionsDropdown isDisabled={!hasSelections}>
                            <DropdownItem
                                key="delete"
                                component="button"
                                onClick={onDeleteSelected}
                            >
                                Delete {numSelected} {pluralize('report', numSelected)}
                            </DropdownItem>
                        </BulkActionsDropdown>
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                        <Pagination
                            itemCount={reportCount}
                            page={currentPage}
                            onSetPage={changePage}
                            perPage={perPage}
                            onPerPageSelect={changePerPage}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            {Object.keys(searchFilter).length !== 0 && (
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem>
                            <SearchFilterResults
                                searchFilter={searchFilter}
                                setSearchFilter={setSearchFilter}
                            />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
            )}
            <Divider component="div" />
            <PageSection isFilled hasOverflowScroll>
                <TableComposable variant="compact">
                    <Thead>
                        <Tr>
                            <Th
                                select={{
                                    onSelect: onSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            {columns.map(({ Header, sortField }, idx) => {
                                const sortParams =
                                    sortField && Boolean(reports.length)
                                        ? {
                                              sort: {
                                                  sortBy: {
                                                      index: activeSortIndex,
                                                      direction: activeSortDirection,
                                                  },
                                                  onSort,
                                                  columnIndex: idx,
                                              },
                                          }
                                        : {};
                                return (
                                    <Th key={Header} modifier="wrap" {...sortParams}>
                                        {Header}
                                    </Th>
                                );
                            })}
                            <Td />
                        </Tr>
                    </Thead>
                    <Tbody>
                        {reports.map((report, rowIndex) => {
                            const { id } = report;

                            const actionItems: ActionItem[] = [];
                            if (hasWriteAccessForReport) {
                                actionItems.push({
                                    title: (
                                        <Button
                                            variant={ButtonVariant.link}
                                            isInline
                                            component={LinkShim}
                                            href={`${vulnManagementReportsPath}/${id}?action=edit`}
                                        >
                                            Edit report
                                        </Button>
                                    ),
                                    onClick: () => {},
                                });

                                // Run option comes second
                                actionItems.push({
                                    title: <div>Run report now</div>,
                                    onClick: () => onClickRun([report.id]),
                                });

                                actionItems.push({
                                    title: (
                                        <div className="pf-u-danger-color-100">Delete report</div>
                                    ),
                                    onClick: () => onClickDelete([report.id]),
                                });
                            }

                            return (
                                <Tr key={id}>
                                    <Td
                                        key={id}
                                        select={{
                                            rowIndex,
                                            onSelect,
                                            isSelected: selected[rowIndex],
                                        }}
                                    />
                                    {columns.map((column) => {
                                        return (
                                            <TableCell
                                                key={column.Header}
                                                row={report}
                                                column={column}
                                            />
                                        );
                                    })}
                                    <Td
                                        actions={{
                                            items: actionItems,
                                        }}
                                    />
                                </Tr>
                            );
                        })}
                        {!reports.length && (
                            <Tr>
                                <Td colSpan={8}>
                                    <Bullseye>
                                        <EmptyStateTemplate
                                            title="No results found"
                                            headingLevel="h2"
                                            icon={SearchIcon}
                                        >
                                            Try clearing some of the filters
                                        </EmptyStateTemplate>
                                    </Bullseye>
                                </Td>
                            </Tr>
                        )}
                    </Tbody>
                </TableComposable>
            </PageSection>
            <ConfirmationModal
                ariaLabel="Confirm deleting reports"
                confirmText="Delete"
                isOpen={deletingReportIds.length > 0}
                onConfirm={onConfirmDeletingReportIds}
                onCancel={onCancelDeleteReportIds}
            >
                {deleteConfirmationText}
            </ConfirmationModal>
        </>
    );
}

export default ReportingTablePanel;
