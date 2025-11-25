import { Divider, ToolbarItem } from '@patternfly/react-core';

import type useURLSort from 'hooks/useURLSort';
import type useURLPagination from 'hooks/useURLPagination';

import { getTableUIState } from 'utils/getTableUIState';
import type { SearchFilter } from 'types/search';
import { overrideManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import ImageOverviewTable, { defaultColumns, tableId } from '../Tables/ImageOverviewTable';
import type { ImageOverviewTableProps } from '../Tables/ImageOverviewTable';
import type { VulnerabilitySeverityLabel } from '../../types';
import TableEntityToolbar from '../../components/TableEntityToolbar';
import type { TableEntityToolbarProps } from '../../components/TableEntityToolbar';
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
    imageTableColumnOverrides: ColumnConfigOverrides<keyof typeof defaultColumns>;
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
    imageTableColumnOverrides,
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

    const columnConfig = overrideManagedColumns(
        managedColumnState.columns,
        imageTableColumnOverrides
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
                <ToolbarItem align={{ default: 'alignEnd' }}>
                    <ColumnManagementButton
                        columnConfig={columnConfig}
                        onApplyColumns={managedColumnState.setVisibility}
                    />
                </ToolbarItem>
            </TableEntityToolbar>
            <Divider component="div" />
            <div
                style={{ overflowX: 'auto' }}
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
