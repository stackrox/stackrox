import React, { useState, useCallback } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { useQuery } from '@apollo/client';

import { PageBody } from 'Components/Panel';
import SidePanelAdjacentArea from 'Components/SidePanelAdjacentArea';
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
                        setSelectedDeploymentId={setSelectedDeploymentId}
                        isViewFiltered={isViewFiltered}
                        setIsViewFiltered={setIsViewFiltered}
                        searchOptions={searchOptions}
                    />
                </div>
                {deploymentId && (
                    <SidePanelAdjacentArea width="3/5">
                        <RiskSidePanel
                            selectedDeploymentId={deploymentId}
                            setSelectedDeploymentId={setSelectedDeploymentId}
                        />
                    </SidePanelAdjacentArea>
                )}
            </PageBody>
        </workflowStateContext.Provider>
    );
};

RiskPage.propTypes = {
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
};

export default RiskPage;
