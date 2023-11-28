import React, { ReactElement, useCallback } from 'react';
import { useHistory, useLocation, useParams } from 'react-router-dom';
import { useQuery } from '@apollo/client';
import { Button } from '@patternfly/react-core';

import PageHeader from 'Components/PageHeader';
import LinkShim from 'Components/PatternFly/LinkShim';
import SearchFilterInput from 'Components/SearchFilterInput';
import entityTypes, { searchCategories } from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import usePermissions from 'hooks/usePermissions';
import useURLSearch from 'hooks/useURLSearch';
import { clustersDelegatedScanningPath } from 'routePaths';
import { Cluster } from 'services/ClustersService';
import parseURL from 'utils/URLParser';

import ClustersTablePanel from './ClustersTablePanel';
import ClustersSidePanel from './ClustersSidePanel';
import ManageTokensButton from './Components/ManageTokensButton';

function ClustersPage(): ReactElement {
    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasReadAccessForDelegatedScanning = hasReadAccess('Administration');
    const hasWriteAccessForIntegration = hasReadWriteAccess('Integration');

    const history = useHistory();
    const { pathname, search } = useLocation();
    const { clusterId: selectedClusterId } = useParams(); // see routePaths for parameter

    const { searchFilter, setSearchFilter } = useURLSearch();
    const workflowState = parseURL({ pathname, search });

    // Handle changes to the currently selected deployment.
    const setSelectedClusterId = useCallback(
        (newCluster: string | Cluster) => {
            const newClusterId = typeof newCluster === 'string' ? newCluster : newCluster?.id ?? '';
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
                {hasReadAccessForDelegatedScanning && (
                    <div className="flex items-center ml-4 mr-1">
                        <Button
                            variant="secondary"
                            component={LinkShim}
                            href={clustersDelegatedScanningPath}
                        >
                            Manage delegated scanning
                        </Button>
                    </div>
                )}
                {hasWriteAccessForIntegration && (
                    <div className="flex items-center ml-1">
                        <Button variant="tertiary">
                            <ManageTokensButton />
                        </Button>
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
}

export default ClustersPage;
