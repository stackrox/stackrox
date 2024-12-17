import React, { ReactElement } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery } from '@apollo/client';

import { searchCategories } from 'constants/entityTypes';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';

import ClustersTablePanel from './ClustersTablePanel';
import ClusterPage from './ClusterPage';

function ClustersPage(): ReactElement {
    const { clusterId } = useParams() as { clusterId: string }; // see routePaths for parameter

    const searchQueryOptions = {
        variables: {
            categories: [searchCategories.CLUSTER],
        },
    };
    const { data: searchData } = useQuery(SEARCH_OPTIONS_QUERY, searchQueryOptions);
    const searchOptions = (searchData && searchData.searchOptions) || [];

    if (clusterId) {
        return <ClusterPage clusterId={clusterId} />;
    }

    return (
        <section className="flex flex-1 flex-col h-full">
            <ClustersTablePanel selectedClusterId={clusterId} searchOptions={searchOptions} />
        </section>
    );
}

export default ClustersPage;
