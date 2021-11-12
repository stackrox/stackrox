import React, { useState, ReactElement } from 'react';
import {
    Flex,
    FlexItem,
    Divider,
    PageSection,
    Pagination,
    DropdownItem,
} from '@patternfly/react-core';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import pluralize from 'pluralize';

import useTableSelection from 'hooks/useTableSelection';
import { TableColumn, SortDirection } from 'hooks/useTableSort';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import TableCell from 'Components/PatternFly/TableCell';
import { ReportConfiguration } from 'types/report.proto';
import DeleteConfirmation from './Modals/DeleteConfirmation';

export type ActionItem = {
    title: string | ReactElement;
    onClick: (item) => void;
};

type ModalType = 'delete' | null;

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
};

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
}: ReportingTablePanelProps): ReactElement {
    const [modalType, setModalType] = useState<ModalType>(null);

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
        setModalType('delete');
    }

    // Handle closing confirmation modals for bulk actions;
    function cancelModal() {
        setModalType(null);
    }

    // Handle closing confirmation modal and clearing selection;
    function closeModal() {
        setModalType(null);
        onClearAll();
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

    const selectedIds = getSelectedIds();

    return (
        <>
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
            <PageSection isFilled padding={{ default: 'noPadding' }} hasOverflowScroll>
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
            <DeleteConfirmation
                isOpen={modalType === 'delete'}
                selectedReportIds={selectedIds}
                closeModal={closeModal}
                cancelModal={cancelModal}
            />
        </>
    );
}

export default ReportingTablePanel;
