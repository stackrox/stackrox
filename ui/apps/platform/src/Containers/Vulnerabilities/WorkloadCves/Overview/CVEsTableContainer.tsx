import React from 'react';
import { useQuery } from '@apollo/client';
import { Bullseye, Spinner } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import CVEsTable, { cveListQuery } from '../Tables/CVEsTable';
import TableErrorComponent from '../components/TableErrorComponent';
import { parseQuerySearchFilter } from '../searchUtils';

const defaultSortFields = ['Deployment', 'Cluster', 'Namespace'];

function CVEsTableContainer() {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage, setPage } = useURLPagination(25);
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'CVE',
            direction: 'asc',
        },
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

    const tableData = data ?? previousData;
    return (
        <>
            {loading && !tableData && (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            )}
            {error && (
                <TableErrorComponent error={error} message="Adjust your filters and try again" />
            )}
            {tableData && (
                <CVEsTable
                    cves={data.imageCVEs}
                    getSortParams={getSortParams}
                    isFiltered={isFiltered}
                />
            )}
        </>
    );
}

export default CVEsTableContainer;
