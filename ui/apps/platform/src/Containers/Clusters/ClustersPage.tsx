import React, { ReactElement } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery } from '@apollo/client';

import { searchCategories } from 'constants/entityTypes';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';

import ClustersTablePanel from './ClustersTablePanel';
import ClustersSidePanel from './ClustersSidePanel';

function ClustersPage(): ReactElement {
    const { clusterId } = useParams(); // see routePaths for parameter

    const searchQueryOptions = {
        variables: {
            categories: [searchCategories.CLUSTER],
        },
    };
    const { data: searchData } = useQuery(SEARCH_OPTIONS_QUERY, searchQueryOptions);
    const searchOptions = (searchData && searchData.searchOptions) || [];

    return (
        <section className="flex flex-1 flex-col h-full">
            <div className="flex flex-1 relative">
                <ClustersTablePanel selectedClusterId={clusterId} searchOptions={searchOptions} />
                {clusterId && <ClustersSidePanel selectedClusterId={clusterId} />}
            </div>
        </section>
    );
}

export default ClustersPage;
