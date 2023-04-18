import React from 'react';
import { useQuery } from '@apollo/client';
import { Bullseye, Spinner } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import ImagesTable, { imageListQuery } from '../Tables/ImagesTable';
import TableErrorComponent from '../components/TableErrorComponent';
import { parseQuerySearchFilter } from '../searchUtils';

const defaultSortFields = ['Image', 'Operating system', 'Deployment count', 'Age', 'Scan time'];

function ImagesTableContainer() {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage, setPage } = useURLPagination(25);
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'Image',
            direction: 'desc',
        },
        onSort: () => setPage(1),
    });

    const { error, loading, data } = useQuery(imageListQuery, {
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

    return (
        <>
            {loading && (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            )}
            {error && (
                <TableErrorComponent error={error} message="Adjust your filters and try again" />
            )}
            {data && (
                <ImagesTable
                    images={data.images}
                    getSortParams={getSortParams}
                    isFiltered={isFiltered}
                />
            )}
        </>
    );
}

export default ImagesTableContainer;
