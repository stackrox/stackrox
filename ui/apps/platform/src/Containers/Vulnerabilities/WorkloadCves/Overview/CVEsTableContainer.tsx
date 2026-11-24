import { useCallback } from 'react';
import { useQuery } from '@apollo/client';
import { DropdownItem, ToolbarItem } from '@patternfly/react-core';

import MenuDropdown from 'Components/PatternFly/MenuDropdown';
import type useURLSort from 'hooks/useURLSort';
import type useURLPagination from 'hooks/useURLPagination';
import useMap from 'hooks/useMap';
import type { VulnerabilityState } from 'types/cve.proto';

import { getTableUIState } from 'utils/getTableUIState';
import type { SearchFilter } from 'types/search';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import { overrideManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import { fetchCVEDetail } from 'services/VulnMgmtService';
import useInvalidateVulnerabilityQueries from '../../hooks/useInvalidateVulnerabilityQueries';
import WorkloadCVEOverviewTable, {
    defaultColumns,
    tableId,
    unfilteredImageCountQuery,
} from '../Tables/WorkloadCVEOverviewTable';
import TableEntityToolbar from '../../components/TableEntityToolbar';
import type { TableEntityToolbarProps } from '../../components/TableEntityToolbar';
import ExceptionRequestModal from '../../components/ExceptionRequestModal/ExceptionRequestModal';
import type { ExceptionRequestModalProps } from '../../components/ExceptionRequestModal/ExceptionRequestModal';
import CompletedExceptionRequestModal from '../../components/ExceptionRequestModal/CompletedExceptionRequestModal';
import useExceptionRequestModal from '../../hooks/useExceptionRequestModal';
import { useVulnMgmtCVEs } from './useVulnMgmtCVEs';

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

    const { error, loading, data } = useVulnMgmtCVEs({
        query: workloadCvesScopedQueryString,
        pagination,
        sortOption,
    });

    const { data: imageCountData } = useQuery(unfilteredImageCountQuery);

    const { invalidateAll: refetchAll } = useInvalidateVulnerabilityQueries();

    const managedColumnState = useManagedColumns(tableId, defaultColumns);
    const selectedCves = useMap<string, ExceptionRequestModalProps['cves'][number]>();

    const resolveSummary = useCallback(
        (cve: string) =>
            fetchCVEDetail(cve, workloadCvesScopedQueryString).then(
                (data) => data.distroDetails?.[0]?.summary ?? ''
            ),
        [workloadCvesScopedQueryString]
    );

    const {
        exceptionRequestModalOptions,
        completedException,
        showModal,
        closeModals,
        createExceptionModalActions,
    } = useExceptionRequestModal({ resolveSummary });

    async function showBulkModal(type: 'DEFERRAL' | 'FALSE_POSITIVE') {
        const cves = await Promise.all(
            Array.from(selectedCves.values()).map(async ({ cve, summary, numAffectedImages }) => ({
                cve,
                summary: summary || (await resolveSummary(cve).catch(() => '')),
                numAffectedImages,
            }))
        );
        showModal({ type, cves });
    }

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
                <ToolbarItem align={{ default: 'alignEnd' }}>
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
                                onClick={() => showBulkModal('DEFERRAL')}
                            >
                                Defer CVEs
                            </DropdownItem>
                            <DropdownItem
                                key="bulk-mark-false-positive"
                                onClick={() => showBulkModal('FALSE_POSITIVE')}
                            >
                                Mark as false positives
                            </DropdownItem>
                        </MenuDropdown>
                    </ToolbarItem>
                )}
            </TableEntityToolbar>
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
                    selectedCves={selectedCves}
                    vulnerabilityState={vulnerabilityState}
                    createTableActions={createTableActions}
                    onClearFilters={() => {
                        onFilterChange({});
                        pagination.setPage(1);
                    }}
                    columnVisibilityState={columnConfig}
                    queryString={workloadCvesScopedQueryString}
                />
            </div>
        </>
    );
}

export default CVEsTableContainer;
