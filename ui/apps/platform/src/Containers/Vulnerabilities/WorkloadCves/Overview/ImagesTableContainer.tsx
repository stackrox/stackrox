import React from 'react';
import { useQuery } from '@apollo/client';
import { Divider } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';

import { getTableUIState } from 'utils/getTableUIState';
import { getPaginationParams } from 'utils/searchUtils';
import ImagesTable, { Image, ImagesTableProps, imageListQuery } from '../Tables/ImagesTable';
import { VulnerabilitySeverityLabel } from '../../types';
import TableEntityToolbar, { TableEntityToolbarProps } from '../../components/TableEntityToolbar';

export { imageListQuery } from '../Tables/ImagesTable';

type ImagesTableContainerProps = {
    filterToolbar: TableEntityToolbarProps['filterToolbar'];
    entityToggleGroup: TableEntityToolbarProps['entityToggleGroup'];
    rowCount: number;
    pagination: ReturnType<typeof useURLPagination>;
    sort: ReturnType<typeof useURLSort>;
    workloadCvesScopedQueryString: string;
    isFiltered: boolean;
    hasWriteAccessForWatchedImage: boolean;
    onWatchImage: ImagesTableProps['onWatchImage'];
    onUnwatchImage: ImagesTableProps['onUnwatchImage'];
    showCveDetailFields: boolean;
};

function ImagesTableContainer({
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
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { page, perPage } = pagination;
    const { sortOption, getSortParams } = sort;

    const { error, loading, data } = useQuery<{
        images: Image[];
    }>(imageListQuery, {
        variables: {
            query: workloadCvesScopedQueryString,
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
    });

    const tableState = getTableUIState({
        isLoading: loading,
        error,
        data: data?.images,
        searchFilter,
    });

    return (
        <>
            <TableEntityToolbar
                filterToolbar={filterToolbar}
                entityToggleGroup={entityToggleGroup}
                pagination={pagination}
                tableRowCount={rowCount}
                isFiltered={isFiltered}
            />
            <Divider component="div" />
            <div
                className="workload-cves-table-container"
                aria-live="polite"
                aria-busy={loading ? 'true' : 'false'}
            >
                <ImagesTable
                    tableState={tableState}
                    getSortParams={getSortParams}
                    isFiltered={isFiltered}
                    filteredSeverities={searchFilter.SEVERITY as VulnerabilitySeverityLabel[]}
                    hasWriteAccessForWatchedImage={hasWriteAccessForWatchedImage}
                    onWatchImage={onWatchImage}
                    onUnwatchImage={onUnwatchImage}
                    showCveDetailFields={showCveDetailFields}
                    onClearFilters={() => setSearchFilter({})}
                />
            </div>
        </>
    );
}

export default ImagesTableContainer;
