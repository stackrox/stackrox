import React from 'react';
import { useQuery } from '@apollo/client';
import { Bullseye, Spinner } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import DeploymentsTable, { Deployment, deploymentListQuery } from '../Tables/DeploymentsTable';
import TableErrorComponent from '../components/TableErrorComponent';
import { parseQuerySearchFilter } from '../searchUtils';

const defaultSortFields = ['Deployment', 'Cluster', 'Namespace'];

function DeploymentsTableContainer() {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage, setPage } = useURLPagination(25);
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'Deployment',
            direction: 'asc',
        },
        onSort: () => setPage(1),
    });

    const { error, loading, data, previousData } = useQuery<{
        deployments: Deployment[];
    }>(deploymentListQuery, {
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
                <DeploymentsTable
                    deployments={tableData.deployments}
                    getSortParams={getSortParams}
                    isFiltered={isFiltered}
                />
            )}
        </>
    );
}

export default DeploymentsTableContainer;
