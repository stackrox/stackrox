import React from 'react';
import { Divider, ToolbarItem } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';

import { getTableUIState } from 'utils/getTableUIState';
import { SearchFilter } from 'types/search';
import { overrideManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import DeploymentsTable, { defaultColumns, tableId } from '../Tables/DeploymentOverviewTable';
import TableEntityToolbar, { TableEntityToolbarProps } from '../../components/TableEntityToolbar';
import { VulnerabilitySeverityLabel } from '../../types';
import { useDeployments } from './useDeployments';

type DeploymentsTableContainerProps = {
    searchFilter: SearchFilter;
    onFilterChange: (searchFilter: SearchFilter) => void;
    filterToolbar: TableEntityToolbarProps['filterToolbar'];
    entityToggleGroup: TableEntityToolbarProps['entityToggleGroup'];
    rowCount: number;
    pagination: ReturnType<typeof useURLPagination>;
    sort: ReturnType<typeof useURLSort>;
    workloadCvesScopedQueryString: string;
    isFiltered: boolean;
    deploymentTableColumnOverrides: ColumnConfigOverrides<keyof typeof defaultColumns>;
};

function DeploymentsTableContainer({
    searchFilter,
    onFilterChange,
    filterToolbar,
    entityToggleGroup,
    rowCount,
    pagination,
    sort,
    workloadCvesScopedQueryString,
    isFiltered,
    deploymentTableColumnOverrides,
}: DeploymentsTableContainerProps) {
    const { sortOption, getSortParams } = sort;

    const { error, loading, data } = useDeployments({
        query: workloadCvesScopedQueryString,
        pagination,
        sortOption,
    });

    const tableState = getTableUIState({
        isLoading: loading,
        error,
        data: data?.deployments,
        searchFilter,
    });

    const managedColumnState = useManagedColumns(tableId, defaultColumns);

    const columnConfig = overrideManagedColumns(
        managedColumnState.columns,
        deploymentTableColumnOverrides
    );

    return (
        <>
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
            </TableEntityToolbar>
            <Divider component="div" />
            <div
                className="workload-cves-table-container"
                aria-live="polite"
                aria-busy={loading ? 'true' : 'false'}
            >
                <DeploymentsTable
                    tableState={tableState}
                    getSortParams={getSortParams}
                    isFiltered={isFiltered}
                    filteredSeverities={searchFilter.SEVERITY as VulnerabilitySeverityLabel[]}
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

export default DeploymentsTableContainer;
