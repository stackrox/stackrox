import { useState } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import { useQuery } from '@apollo/client';

import { PageBody } from 'Components/Panel';
import { searchCategories } from 'constants/entityTypes';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import RiskPageHeader from './RiskPageHeader';
import RiskTablePanel from './RiskTablePanel';

function RiskTablePage() {
    const params = useParams();
    const { deploymentId } = params;

    // Handle changes to applied search options.
    const [isViewFiltered, setIsViewFiltered] = useState(false);

    const searchQueryOptions = {
        variables: {
            categories: [searchCategories.DEPLOYMENT],
        },
    };
    const { data: searchData } = useQuery(SEARCH_OPTIONS_QUERY, searchQueryOptions);
    const searchOptions = searchData?.searchOptions ?? [];
    const filteredSearchOptions = searchOptions.filter(
        (option) => option !== 'Orchestrator Component'
    );
    return (
        <>
            <RiskPageHeader isViewFiltered={isViewFiltered} searchOptions={filteredSearchOptions} />
            <PageBody>
                <div className="flex-shrink-1 overflow-hidden w-full">
                    <RiskTablePanel
                        selectedDeploymentId={deploymentId}
                        isViewFiltered={isViewFiltered}
                        setIsViewFiltered={setIsViewFiltered}
                    />
                </div>
            </PageBody>
        </>
    );
}

export default RiskTablePage;
