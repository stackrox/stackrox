import React from 'react';
import { useQuery } from '@apollo/client';
import { Bullseye, Spinner, Divider } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import CVEsTable, { cveListQuery, unfilteredImageCountQuery } from '../Tables/CVEsTable';
import TableErrorComponent from '../components/TableErrorComponent';
import { EntityCounts } from '../components/EntityTypeToggleGroup';
import { DefaultFilters } from '../types';
import { parseQuerySearchFilter } from '../searchUtils';
import { defaultCVESortFields, CVEsDefaultSort } from '../sortUtils';
import TableEntityToolbar from '../components/TableEntityToolbar';

type CVEsTableContainerProps = {
    defaultFilters: DefaultFilters;
    countsData: EntityCounts;
};

function CVEsTableContainer({ defaultFilters, countsData }: CVEsTableContainerProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const pagination = useURLPagination(20);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams, setSortOption } = useURLSort({
        sortFields: defaultCVESortFields,
        defaultSortOption: CVEsDefaultSort,
        onSort: () => setPage(1),
    });

    const { error, loading, data, previousData } = useQuery(cveListQuery, {
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

    const { data: imageCountData } = useQuery(unfilteredImageCountQuery);

    const tableData = data ?? previousData;
    return (
        <>
            <TableEntityToolbar
                defaultFilters={defaultFilters}
                countsData={countsData}
                setSortOption={setSortOption}
                pagination={pagination}
                tableRowCount={countsData.imageCVECount}
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
                    <CVEsTable
                        cves={tableData.imageCVEs}
                        unfilteredImageCount={imageCountData?.imageCount || 0}
                        getSortParams={getSortParams}
                        isFiltered={isFiltered}
                    />
                </div>
            )}
        </>
    );
}

export default CVEsTableContainer;
