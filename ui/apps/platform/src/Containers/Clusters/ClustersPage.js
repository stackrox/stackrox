import React, { useCallback } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { useQuery } from '@apollo/client';

import PageHeader from 'Components/PageHeader';
import SearchFilterInput from 'Components/SearchFilterInput';
import entityTypes, { searchCategories } from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import usePermissions from 'hooks/usePermissions';
import useURLSearch from 'hooks/useURLSearch';
import parseURL from 'utils/URLParser';

import ClustersTablePanel from './ClustersTablePanel';
import ClustersSidePanel from './ClustersSidePanel';
import ManageTokensButton from './Components/ManageTokensButton';

const ClustersPage = ({
    history,
    location: { pathname, search },
    match: {
        params: { clusterId: selectedClusterId },
    },
}) => {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForIntegration = hasReadWriteAccess('Integration');

    const { searchFilter, setSearchFilter } = useURLSearch();
    const workflowState = parseURL({ pathname, search });

    // Handle changes to the currently selected deployment.
    const setSelectedClusterId = useCallback(
        (newCluster) => {
            const newClusterId = newCluster?.id || newCluster || '';
            const newWorkflowState = newClusterId
                ? workflowState.pushRelatedEntity(entityTypes.CLUSTER, newClusterId)
                : workflowState.pop();

            const newUrl = newWorkflowState.toUrl();

            history.push(newUrl);
        },
        [workflowState, history]
    );

    const searchQueryOptions = {
        variables: {
            categories: [searchCategories.CLUSTER],
        },
    };
    const { data: searchData } = useQuery(SEARCH_OPTIONS_QUERY, searchQueryOptions);
    const searchOptions = (searchData && searchData.searchOptions) || [];

    const headerText = 'Clusters';
    const subHeaderText = 'Resource list';

    const pageHeader = (
        <PageHeader header={headerText} subHeader={subHeaderText}>
            <div className="flex flex-1 items-center justify-end">
                <SearchFilterInput
                    className="w-full"
                    searchFilter={searchFilter}
                    searchOptions={searchOptions}
                    searchCategory="CLUSTERS"
                    placeholder="Filter clusters"
                    handleChangeSearchFilter={setSearchFilter}
                />
                {hasWriteAccessForIntegration && (
                    <div className="flex items-center ml-4 mr-3">
                        <ManageTokensButton />
                    </div>
                )}
            </div>
        </PageHeader>
    );

    return (
        <workflowStateContext.Provider value={workflowState}>
            <section className="flex flex-1 flex-col h-full">
                <div className="flex flex-1 flex-col">
                    {pageHeader}
                    <div className="flex flex-1 relative">
                        <ClustersTablePanel
                            selectedClusterId={selectedClusterId}
                            setSelectedClusterId={setSelectedClusterId}
                            searchOptions={searchOptions}
                        />
                        <ClustersSidePanel
                            selectedClusterId={selectedClusterId}
                            setSelectedClusterId={setSelectedClusterId}
                        />
                    </div>
                </div>
            </section>
        </workflowStateContext.Provider>
    );
};

ClustersPage.propTypes = {
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
};

export default ClustersPage;
