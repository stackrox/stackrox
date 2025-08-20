import React from 'react';
import { Divider, ToolbarItem } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import usePermissions from 'hooks/usePermissions';

import { getTableUIState } from 'utils/getTableUIState';
import { SearchFilter } from 'types/search';
import { hideColumnIf, overrideManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import ImageOverviewTable, {
    ImageOverviewTableProps,
    defaultColumns,
    tableId,
} from '../Tables/ImageOverviewTable';
import { VulnerabilitySeverityLabel } from '../../types';
import TableEntityToolbar, { TableEntityToolbarProps } from '../../components/TableEntityToolbar';
import { useImages } from './useImages';

type ImagesTableContainerProps = {
    searchFilter: SearchFilter;
    onFilterChange: (searchFilter: SearchFilter) => void;
    filterToolbar: TableEntityToolbarProps['filterToolbar'];
    entityToggleGroup: TableEntityToolbarProps['entityToggleGroup'];
    rowCount: number;
    pagination: ReturnType<typeof useURLPagination>;
    sort: ReturnType<typeof useURLSort>;
    workloadCvesScopedQueryString: string;
    isFiltered: boolean;
    hasWriteAccessForWatchedImage: boolean;
    onWatchImage: ImageOverviewTableProps['onWatchImage'];
    onUnwatchImage: ImageOverviewTableProps['onUnwatchImage'];
    showCveDetailFields: boolean;
};

function ImagesTableContainer({
    searchFilter,
    onFilterChange,
    filterToolbar,
    entityToggleGroup,
    rowCount,
    pagination,
    sort,
    workloadCvesScopedQueryString,
    isFiltered,
    hasWriteAccessForWatchedImage,
    onWatchImage,
    onUnwatchImage,
    showCveDetailFields,
}: ImagesTableContainerProps) {
    const { sortOption, getSortParams } = sort;

    const { error, loading, data } = useImages({
        query: workloadCvesScopedQueryString,
        pagination,
        sortOption,
    });

    const tableState = getTableUIState({
        isLoading: loading,
        error,
        data: data?.images,
        searchFilter,
    });

    const managedColumnState = useManagedColumns(tableId, defaultColumns);

    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForImage = hasReadWriteAccess('Image'); // SBOM Generation mutates image scan state.
    const hasActionColumn = hasWriteAccessForWatchedImage || hasWriteAccessForImage;

    const columnConfig = overrideManagedColumns(managedColumnState.columns, {
        cvesBySeverity: hideColumnIf(!showCveDetailFields),
        rowActions: hideColumnIf(!hasActionColumn),
    });

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
                <ImageOverviewTable
                    tableState={tableState}
                    getSortParams={getSortParams}
                    isFiltered={isFiltered}
                    filteredSeverities={searchFilter.SEVERITY as VulnerabilitySeverityLabel[]}
                    hasWriteAccessForWatchedImage={hasWriteAccessForWatchedImage}
                    onWatchImage={onWatchImage}
                    onUnwatchImage={onUnwatchImage}
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

export default ImagesTableContainer;
