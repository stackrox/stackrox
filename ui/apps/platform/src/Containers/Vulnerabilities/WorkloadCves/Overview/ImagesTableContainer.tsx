import React from 'react';
import { useQuery } from '@apollo/client';
import { Bullseye, Spinner, Divider } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import ImagesTable, { imageListQuery } from '../Tables/ImagesTable';
import TableErrorComponent from '../components/TableErrorComponent';
import { EntityCounts } from '../components/EntityTypeToggleGroup';
import { parseQuerySearchFilter } from '../searchUtils';
import { defaultImageSortFields, imagesDefaultSort } from '../sortUtils';
import { DefaultFilters } from '../types';
import TableEntityToolbar from '../components/TableEntityToolbar';

type ImagesTableContainerProps = {
    defaultFilters: DefaultFilters;
    countsData: EntityCounts;
};

function ImagesTableContainer({ defaultFilters, countsData }: ImagesTableContainerProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const pagination = useURLPagination(20);
    const { page, perPage, setPage } = pagination;
    const sort = useURLSort({
        sortFields: defaultImageSortFields,
        defaultSortOption: imagesDefaultSort,
        onSort: () => setPage(1),
    });
    const { sortOption, getSortParams, setSortOption } = sort;

    const { error, loading, data, previousData } = useQuery(imageListQuery, {
        variables: {
            query: getRequestQueryStringForSearchFilter({
                ...querySearchFilter,
            }),
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
            {tableData && (
                <div className="workload-cves-table-container">
                    <ImagesTable
                        images={tableData.images}
                        getSortParams={getSortParams}
                        isFiltered={isFiltered}
                    />
                </div>
            )}
        </>
    );
}

export default ImagesTableContainer;
