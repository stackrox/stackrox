import React from 'react';
import { useQuery } from '@apollo/client';
import { Bullseye, Spinner, Divider } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied } from 'utils/searchUtils';
import ImagesTable, { ImagesTableProps, imageListQuery } from '../Tables/ImagesTable';
import TableErrorComponent from '../components/TableErrorComponent';
import { EntityCounts } from '../components/EntityTypeToggleGroup';
import { getCveStatusScopedQueryString, parseQuerySearchFilter } from '../searchUtils';
import { defaultImageSortFields, imagesDefaultSort } from '../sortUtils';
import { DefaultFilters, VulnerabilitySeverityLabel, CveStatusTab } from '../types';
import TableEntityToolbar from '../components/TableEntityToolbar';

export { imageListQuery } from '../Tables/ImagesTable';

type ImagesTableContainerProps = {
    defaultFilters: DefaultFilters;
    countsData: EntityCounts;
    cveStatusTab?: CveStatusTab; // TODO Make this required once Observed/Deferred/FP states are re-implemented
    pagination: ReturnType<typeof useURLPagination>;
    onWatchImage: ImagesTableProps['onWatchImage'];
    onUnwatchImage: ImagesTableProps['onUnwatchImage'];
};

function ImagesTableContainer({
    defaultFilters,
    countsData,
    cveStatusTab,
    pagination,
    onWatchImage,
    onUnwatchImage,
}: ImagesTableContainerProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage, setPage } = pagination;
    const sort = useURLSort({
        sortFields: defaultImageSortFields,
        defaultSortOption: imagesDefaultSort,
        onSort: () => setPage(1),
    });
    const { sortOption, getSortParams, setSortOption } = sort;

    const { error, loading, data, previousData } = useQuery(imageListQuery, {
        variables: {
            query: getCveStatusScopedQueryString(querySearchFilter, cveStatusTab),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
        },
    });

    const tableData = data ?? previousData;
    return (
        <>
            <TableEntityToolbar
                defaultFilters={defaultFilters}
                countsData={countsData}
                setSortOption={setSortOption}
                pagination={pagination}
                tableRowCount={countsData.imageCount}
                isFiltered={isFiltered}
            />
            <Divider component="div" />
            {loading && !tableData && (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            )}
            {error && (
                <TableErrorComponent error={error} message="Adjust your filters and try again" />
            )}
            {!error && tableData && (
                <div className="workload-cves-table-container">
                    <ImagesTable
                        images={tableData.images}
                        getSortParams={getSortParams}
                        isFiltered={isFiltered}
                        filteredSeverities={searchFilter.Severity as VulnerabilitySeverityLabel[]}
                        onWatchImage={onWatchImage}
                        onUnwatchImage={onUnwatchImage}
                    />
                </div>
            )}
        </>
    );
}

export default ImagesTableContainer;
