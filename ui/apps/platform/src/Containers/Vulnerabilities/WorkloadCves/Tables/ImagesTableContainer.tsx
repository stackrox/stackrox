import React from 'react';
import { gql, useQuery } from '@apollo/client';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import ImagesTable from './ImagesTable';

const imageListQuery = gql`
    query getImageList($query: String, $pagination: Pagination) {
        images(query: $query, pagination: $pagination) {
            id
            name {
                registry
                remote
                tag
            }
            imageCVECountBySeverity(query: $query) {
                critical
                important
                moderate
                low
            }
            operatingSystem
            deploymentCount(query: $query)
            watchStatus
            metadata {
                v1 {
                    created
                }
            }
            scanTime
        }
    }
`;

const defaultSortFields = ['Image', 'Operating system', 'Deployment count', 'Age', 'Scan time'];

function ImagesTableContainer() {
    // TODO: add filter query and pagination from URL
    const { error, loading, data } = useQuery(imageListQuery, {});
    const { setPage } = useURLPagination(50);
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'Severity',
            direction: 'desc',
        },
        onSort: () => setPage(1),
    });
    console.log(data);

    return (
        <>
            {loading && <div>loading</div>}
            {error && <div>error</div>}
            {data && <ImagesTable images={data.images} getSortParams={getSortParams} />}
        </>
    );
}

export default ImagesTableContainer;
