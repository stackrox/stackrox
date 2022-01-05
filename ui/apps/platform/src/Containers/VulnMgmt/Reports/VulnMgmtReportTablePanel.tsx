import React, { useState, ReactElement } from 'react';
import { useSelector } from 'react-redux';
import {
    Alert,
    AlertGroup,
    AlertVariant,
    Flex,
    FlexItem,
    Divider,
    PageSection,
    PageSectionVariants,
    Pagination,
    DropdownItem,
} from '@patternfly/react-core';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import pluralize from 'pluralize';
import { createStructuredSelector } from 'reselect';

import useTableSelection from 'hooks/useTableSelection';
import { TableColumn, SortDirection } from 'hooks/useTableSort';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import TableCell from 'Components/PatternFly/TableCell';
import { selectors } from 'reducers';
import { getHasReadWritePermission } from 'reducers/roles';
import { ReportConfiguration } from 'types/report.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type ActionItem = {
    title: string | ReactElement;
    onClick: (item) => void;
};

type AlertInfo = {
    title: string;
    variant: AlertVariant;
    key: number;
};

type ReportingTablePanelProps = {
    reports: ReportConfiguration[];
    reportCount: number;
    currentPage: number;
    setCurrentPage: (page) => void;
    perPage: number;
    setPerPage: (perPage) => void;
    activeSortIndex: number;
    setActiveSortIndex: (idx) => void;
    activeSortDirection: SortDirection;
    setActiveSortDirection: (dir) => void;
    columns: TableColumn[];
    onDeleteReports: (integration) => Promise<any>;
};

const permissionsSelector = createStructuredSelector({
    userRolePermissions: selectors.getUserRolePermissions,
});

function ReportingTablePanel({
    reports,
    reportCount,
    currentPage,
    setCurrentPage,
    perPage,
    setPerPage,
    activeSortIndex,
    setActiveSortIndex,
    activeSortDirection,
    setActiveSortDirection,
    columns,
    onDeleteReports,
}: ReportingTablePanelProps): ReactElement {
    const [alerts, setAlerts] = React.useState<AlertInfo[]>([]);
    const [deletingReportIds, setDeletingReportIds] = useState<string[]>([]);
    const { userRolePermissions } = useSelector(permissionsSelector);
    const hasWriteAccessForVulnerabilityReports = getHasReadWritePermission(
        'VulnerabilityReports',
        userRolePermissions
    );

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
                    title: message || 'An unknown error occurred while deleting',
                    variant: AlertVariant.danger,
                    key: new Date().getTime(),
                };
                setAlerts((prevAlertInfo) => [...prevAlertInfo, alertInfo]);

                setDeletingReportIds([]);
                throw error;
            });
    }

    function onCancelDeleteReportIds() {
        setDeletingReportIds([]);
    }

    const deleteConfirmationText = `Are you sure you want to delete ${
        deletingReportIds.length
    } ${pluralize('report', deletingReportIds.length)}`;

    return (
        <>
            <PageSection padding={{ default: 'padding' }} variant={PageSectionVariants.light}>
                <AlertGroup
                    isLiveRegion
                    aria-live="polite"
                    aria-relevant="additions text"
                    aria-atomic="false"
                >
                    {alerts.map(({ title, variant, key }) => (
                        <Alert isInline variant={variant} title={title} key={key} />
                    ))}
                </AlertGroup>
            </PageSection>
            <Flex
                className="pf-u-p-md"
                alignSelf={{ default: 'alignSelfCenter' }}
                fullWidth={{ default: 'fullWidth' }}
            >
                <FlexItem data-testid="reports-bulk-actions-dropdown">
                    <BulkActionsDropdown isDisabled={!hasSelections}>
                        <DropdownItem key="delete" component="button" onClick={onDeleteSelected}>
                            Delete {numSelected} {pluralize('report', numSelected)}
                        </DropdownItem>
                    </BulkActionsDropdown>
                </FlexItem>
                <FlexItem align={{ default: 'alignRight' }}>
                    <Pagination
                        itemCount={reportCount}
                        page={currentPage}
                        onSetPage={changePage}
                        perPage={perPage}
                        onPerPageSelect={changePerPage}
                    />
                </FlexItem>
            </Flex>
            <Divider component="div" />
            <PageSection
                isFilled
                padding={{ default: 'noPadding' }}
                hasOverflowScroll
                variant={PageSectionVariants.light}
            >
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
                                const sortParams = sortField
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
                            <Th />
                        </Tr>
                    </Thead>
                    <Tbody>
                        {reports.map((report, rowIndex) => {
                            const { id } = report;

                            const actionItems: ActionItem[] = [];
                            if (hasWriteAccessForVulnerabilityReports) {
                                actionItems.push({
                                    title: (
                                        <div className="pf-u-danger-color-100">Delete report</div>
                                    ),
                                    onClick: () => onClickDelete([report.id]),
                                });
                            }

                            return (
                                // eslint-disable-next-line react/no-array-index-key
                                <Tr key={rowIndex}>
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
                    </Tbody>
                </TableComposable>
            </PageSection>
            <ConfirmationModal
                ariaLabel="Confirm deleting reports"
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
