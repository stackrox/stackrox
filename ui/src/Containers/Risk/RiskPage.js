import React, { useState, useCallback } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { useQuery } from 'react-apollo';

import entityTypes, { searchCategories } from 'constants/entityTypes';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import workflowStateContext from 'Containers/workflowStateContext';
import parseURL from 'utils/URLParser';
import RiskPageHeader from './RiskPageHeader';
import RiskSidePanel from './RiskSidePanel';
import RiskTablePanel from './RiskTablePanel';

const RiskPage = ({
    history,
    location: { pathname, search },
    match: {
        params: { deploymentId },
    },
}) => {
    const workflowState = parseURL({ pathname, search });

    // Handle changes to applied search options.
    const [isViewFiltered, setIsViewFiltered] = useState(false);

    // Handle changes to the currently selected deployment.
    const setSelectedDeploymentId = useCallback(
        (newDeploymentId) => {
            const newWorkflowState = newDeploymentId
                ? workflowState.pushRelatedEntity(entityTypes.DEPLOYMENT, newDeploymentId)
                : workflowState.pop();

            const newUrl = newWorkflowState.toUrl();

            history.push(newUrl);
        },
        [workflowState, history]
    );

    const searchQueryOptions = {
        variables: {
            categories: [searchCategories.DEPLOYMENT],
        },
    };
    const { data: searchData } = useQuery(SEARCH_OPTIONS_QUERY, searchQueryOptions);
    const searchOptions = (searchData && searchData.searchOptions) || [];
    const autoFocusSearchInput = !deploymentId;

    return (
        <workflowStateContext.Provider value={workflowState}>
            <section className="flex flex-1 flex-col h-full">
                <div className="flex flex-1 flex-col">
                    <RiskPageHeader
                        setSelectedDeploymentId={setSelectedDeploymentId}
                        isViewFiltered={isViewFiltered}
                        searchOptions={searchOptions}
                        autoFocusSearchInput={autoFocusSearchInput}
                    />
                    <div className="flex flex-1 relative">
                        <div className="shadow border-primary-300 w-full overflow-hidden">
                            <RiskTablePanel
                                selectedDeploymentId={deploymentId}
                                setSelectedDeploymentId={setSelectedDeploymentId}
                                isViewFiltered={isViewFiltered}
                                setIsViewFiltered={setIsViewFiltered}
                                searchOptions={searchOptions}
                            />
                        </div>
                        <RiskSidePanel
                            selectedDeploymentId={deploymentId}
                            setSelectedDeploymentId={setSelectedDeploymentId}
                        />
                    </div>
                </div>
            </section>
        </workflowStateContext.Provider>
    );
};

RiskPage.propTypes = {
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
};

export default RiskPage;
