import React from 'react';
import { useQuery } from '@apollo/client';
import { Divider, DropdownItem, ToolbarItem } from '@patternfly/react-core';

import MenuDropdown from 'Components/PatternFly/MenuDropdown';
import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useMap from 'hooks/useMap';
import { VulnerabilityState } from 'types/cve.proto';

import { getTableUIState } from 'utils/getTableUIState';
import useHasRequestExceptionsAbility from 'Containers/Vulnerabilities/hooks/useHasRequestExceptionsAbility';
import { SearchFilter } from 'types/search';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { hideColumnIf, overrideManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
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

    const hasRequestExceptionsAbility = useHasRequestExceptionsAbility();

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNvdCvssColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const isEpssProbabilityColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const managedColumnState = useManagedColumns(tableId, defaultColumns);
    const selectedCves = useMap<string, ExceptionRequestModalProps['cves'][number]>();
    const {
        exceptionRequestModalOptions,
        completedException,
        showModal,
        closeModals,
        createExceptionModalActions,
    } = useExceptionRequestModal();
    const showDeferralUI = hasRequestExceptionsAbility && vulnerabilityState === 'OBSERVED';
    const canSelectRows = showDeferralUI;

    const createTableActions = showDeferralUI ? createExceptionModalActions : undefined;

    const columnConfig = overrideManagedColumns(managedColumnState.columns, {
        cveSelection: hideColumnIf(!canSelectRows),
        topNvdCvss: hideColumnIf(!isNvdCvssColumnEnabled),
        epssProbability: hideColumnIf(!isEpssProbabilityColumnEnabled),
        requestDetails: hideColumnIf(vulnerabilityState === 'OBSERVED'),
        rowActions: hideColumnIf(createTableActions === undefined),
    });

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
                {canSelectRows && (
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
                className="workload-cves-table-container"
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
