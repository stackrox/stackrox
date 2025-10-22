import React from 'react';
import { useQuery } from '@apollo/client';
import { Divider, DropdownItem, ToolbarItem } from '@patternfly/react-core';

import MenuDropdown from 'Components/PatternFly/MenuDropdown';
import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useMap from 'hooks/useMap';
import { VulnerabilityState } from 'types/cve.proto';

import { getTableUIState } from 'utils/getTableUIState';
import { SearchFilter } from 'types/search';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import { overrideManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import useInvalidateVulnerabilityQueries from '../../hooks/useInvalidateVulnerabilityQueries';
import WorkloadCVEOverviewTable, {
    defaultColumns,
    tableId,
    unfilteredImageCountQuery,
} from '../Tables/WorkloadCVEOverviewTable';
import { VulnerabilitySeverityLabel } from '../../types';
import TableEntityToolbar, { TableEntityToolbarProps } from '../../components/TableEntityToolbar';
import ExceptionRequestModal, {
    ExceptionRequestModalProps,
} from '../../components/ExceptionRequestModal/ExceptionRequestModal';
import CompletedExceptionRequestModal from '../../components/ExceptionRequestModal/CompletedExceptionRequestModal';
import useExceptionRequestModal from '../../hooks/useExceptionRequestModal';
import { useImageCves } from './useImageCves';

export type CVEsTableContainerProps = {
    searchFilter: SearchFilter;
    onFilterChange: (searchFilter: SearchFilter) => void;
    filterToolbar: TableEntityToolbarProps['filterToolbar'];
    entityToggleGroup: TableEntityToolbarProps['entityToggleGroup'];
    rowCount: number;
    vulnerabilityState: VulnerabilityState;
    pagination: ReturnType<typeof useURLPagination>;
    sort: ReturnType<typeof useURLSort>;
    workloadCvesScopedQueryString: string;
    isFiltered: boolean;
    showDeferralUI: boolean;
    cveTableColumnOverrides: ColumnConfigOverrides<keyof typeof defaultColumns>;
};

function CVEsTableContainer({
    searchFilter,
    onFilterChange,
    filterToolbar,
    entityToggleGroup,
    rowCount,
    vulnerabilityState,
    pagination,
    sort,
    workloadCvesScopedQueryString,
    isFiltered,
    showDeferralUI,
    cveTableColumnOverrides,
}: CVEsTableContainerProps) {
    const { sortOption, getSortParams } = sort;

    const { error, loading, data } = useImageCves({
        query: workloadCvesScopedQueryString,
        pagination,
        sortOption,
        vulnerabilityState,
    });

    const { data: imageCountData } = useQuery(unfilteredImageCountQuery);

    const { invalidateAll: refetchAll } = useInvalidateVulnerabilityQueries();

    const managedColumnState = useManagedColumns(tableId, defaultColumns);
    const selectedCves = useMap<string, ExceptionRequestModalProps['cves'][number]>();
    const {
        exceptionRequestModalOptions,
        completedException,
        showModal,
        closeModals,
        createExceptionModalActions,
    } = useExceptionRequestModal();

    const createTableActions = showDeferralUI ? createExceptionModalActions : undefined;

    const columnConfig = overrideManagedColumns(
        managedColumnState.columns,
        cveTableColumnOverrides
    );

    const tableState = getTableUIState({
        isLoading: loading,
        data: data?.imageCVEs,
        error,
        searchFilter,
    });

    return (
        <>
            {exceptionRequestModalOptions && (
                <ExceptionRequestModal
                    cves={exceptionRequestModalOptions.cves}
                    type={exceptionRequestModalOptions.type}
                    scopeContext="GLOBAL"
                    onExceptionRequestSuccess={(exception) => {
                        selectedCves.clear();
                        showModal({ type: 'COMPLETION', exception });
                        return refetchAll();
                    }}
                    onClose={closeModals}
                />
            )}
            {completedException && (
                <CompletedExceptionRequestModal
                    exceptionRequest={completedException}
                    onClose={closeModals}
                />
            )}
            <TableEntityToolbar
                filterToolbar={filterToolbar}
                entityToggleGroup={entityToggleGroup}
                pagination={pagination}
                tableRowCount={rowCount}
                isFiltered={isFiltered}
            >
                <ToolbarItem align={{ default: 'alignRight' }}>
                    <ColumnManagementButton
                        columnConfig={columnConfig}
                        onApplyColumns={managedColumnState.setVisibility}
                    />
                </ToolbarItem>
                {showDeferralUI && (
                    <ToolbarItem>
                        <MenuDropdown
                            toggleText="Bulk actions"
                            isDisabled={selectedCves.size === 0}
                        >
                            <DropdownItem
                                key="bulk-defer-cve"
                                onClick={() =>
                                    showModal({
                                        type: 'DEFERRAL',
                                        cves: Array.from(selectedCves.values()),
                                    })
                                }
                            >
                                Defer CVEs
                            </DropdownItem>
                            <DropdownItem
                                key="bulk-mark-false-positive"
                                onClick={() =>
                                    showModal({
                                        type: 'FALSE_POSITIVE',
                                        cves: Array.from(selectedCves.values()),
                                    })
                                }
                            >
                                Mark as false positives
                            </DropdownItem>
                        </MenuDropdown>
                    </ToolbarItem>
                )}
            </TableEntityToolbar>
            <Divider component="div" />
            <div
                style={{ overflowX: 'auto' }}
                aria-live="polite"
                aria-busy={loading ? 'true' : 'false'}
            >
                <WorkloadCVEOverviewTable
                    tableState={tableState}
                    unfilteredImageCount={imageCountData?.imageCount || 0}
                    getSortParams={getSortParams}
                    isFiltered={isFiltered}
                    filteredSeverities={searchFilter.SEVERITY as VulnerabilitySeverityLabel[]}
                    selectedCves={selectedCves}
                    vulnerabilityState={vulnerabilityState}
                    createTableActions={createTableActions}
                    onClearFilters={() => {
                        onFilterChange({});
                        pagination.setPage(1);
                    }}
                    columnVisibilityState={columnConfig}
                />
            </div>
        </>
    );
}

export default CVEsTableContainer;
