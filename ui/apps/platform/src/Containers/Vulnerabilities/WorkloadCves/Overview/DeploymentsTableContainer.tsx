import { ToolbarItem } from '@patternfly/react-core';

import useDeploymentStatus from 'hooks/useDeploymentStatus';
import { getDeploymentStatusQueryString } from '../../utils/searchUtils';
import DeploymentStatusFilter from '../components/DeploymentStatusFilter';

import useFeatureFlags from 'hooks/useFeatureFlags';

import type useURLSort from 'hooks/useURLSort';
import type useURLPagination from 'hooks/useURLPagination';

import { getTableUIState } from 'utils/getTableUIState';
import type { SearchFilter } from 'types/search';
import { overrideManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import DeploymentsTable, { defaultColumns, tableId } from '../Tables/DeploymentOverviewTable';
import TableEntityToolbar from '../../components/TableEntityToolbar';
import type { TableEntityToolbarProps } from '../../components/TableEntityToolbar';
import type { VulnerabilitySeverityLabel } from '../../types';
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isTombstonesEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_TOMBSTONES');

    const deploymentStatus = useDeploymentStatus();

    const { sortOption, getSortParams } = sort;

    const deploymentsQueryString = getDeploymentStatusQueryString(
        workloadCvesScopedQueryString,
        isTombstonesEnabled ? deploymentStatus : 'DEPLOYED'
    );

    const { error, loading, data } = useDeployments({
        query: deploymentsQueryString,
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
                {isTombstonesEnabled && (
                    <ToolbarItem>
                        <DeploymentStatusFilter onChange={() => pagination.setPage(1)} />
                    </ToolbarItem>
                )}
                <ToolbarItem align={{ default: 'alignEnd' }}>
                    <ColumnManagementButton
                        columnConfig={columnConfig}
                        onApplyColumns={managedColumnState.setVisibility}
                    />
                </ToolbarItem>
            </TableEntityToolbar>
            <div
                style={{ overflowX: 'auto' }}
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
