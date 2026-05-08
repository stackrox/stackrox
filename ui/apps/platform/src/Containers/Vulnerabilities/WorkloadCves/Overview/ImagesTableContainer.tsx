import { ToolbarItem } from '@patternfly/react-core';

import type useURLSort from 'hooks/useURLSort';
import type useURLPagination from 'hooks/useURLPagination';

import { getTableUIState } from 'utils/getTableUIState';
import type { SearchFilter } from 'types/search';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { overrideManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import ImageOverviewTable, { defaultColumns, tableId } from '../Tables/ImageOverviewTable';
import type { ImageOverviewTableProps } from '../Tables/ImageOverviewTable';
import type { VulnerabilitySeverityLabel } from '../../types';
import TableEntityToolbar from '../../components/TableEntityToolbar';
import type { TableEntityToolbarProps } from '../../components/TableEntityToolbar';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
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
    const { viewContext } = useWorkloadCveViewContext();
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const { error, loading, data } = useImages({
        query: workloadCvesScopedQueryString,
        pagination,
        sortOption,
    });

    // When viewing inactive images with soft deletion enabled, exclude images that
    // still have active deployments. This handles an edge case where an image is
    // associated with both active and deleted deployments: the server-side query
    // returns the image (because the deleted deployment row passes the filter), but
    // it should not appear in the inactive tab.
    const isInactiveWithSoftDeletion =
        viewContext === 'Inactive images' && isFeatureFlagEnabled('ROX_DEPLOYMENT_SOFT_DELETION');
    const images = isInactiveWithSoftDeletion
        ? data?.images.filter((image) => image.activeDeploymentCount === 0)
        : data?.images;

    const tableState = getTableUIState({
        isLoading: loading,
        error,
        data: images,
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
