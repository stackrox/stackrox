import { useState } from 'react';
import { useLocation, useParams } from 'react-router-dom-v5-compat';
import { useQuery } from '@apollo/client';

import { PageBody } from 'Components/Panel';
import { searchCategories } from 'constants/entityTypes';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import workflowStateContext from 'Containers/workflowStateContext';
import parseURL from 'utils/URLParser';
import RiskPageHeader from './RiskPageHeader';
import RiskTablePanel from './RiskTablePanel';

function RiskTablePage() {
    const location = useLocation();
    const params = useParams();
    const { deploymentId } = params;
    const { pathname, search } = location;
    const workflowState = parseURL({ pathname, search });

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
        <workflowStateContext.Provider value={workflowState}>
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
        </workflowStateContext.Provider>
    );
}

export default RiskTablePage;
