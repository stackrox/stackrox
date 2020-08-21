import React, { useEffect, useState, useCallback } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { generatePath } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { useQuery } from '@apollo/client';

import PageHeader from 'Components/PageHeader';
import ToggleSwitch from 'Components/ToggleSwitch';
import URLSearchInput from 'Components/URLSearchInput';
import entityTypes, { searchCategories } from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import { actions as clustersActions } from 'reducers/clusters';
import { selectors } from 'reducers';
import { clustersPath } from 'routePaths';
import { getAutoUpgradeConfig, saveAutoUpgradeConfig } from 'services/ClustersService';
import parseURL from 'utils/URLParser';

import ClustersTablePanel from './ClustersTablePanel';
import ClustersSidePanel from './ClustersSidePanel';
// @TODO, refactor these helper utilities to this folder,
//        when retiring clusters in Integrations section

const ClustersPage = ({
    history,
    location: { pathname, search },
    match: {
        params: { clusterId: selectedClusterId },
    },
}) => {
    const workflowState = parseURL({ pathname, search });

    const [autoUpgradeConfig, setAutoUpgradeConfig] = useState({});

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
    const autoFocusSearchInput = !selectedClusterId;

    function fetchConfig() {
        getAutoUpgradeConfig().then((config) => {
            setAutoUpgradeConfig(config);
        });
    }

    useEffect(() => {
        fetchConfig();
    }, []);

    // When the selected cluster changes, update the URL.
    useEffect(() => {
        const newPath = selectedClusterId
            ? generatePath(clustersPath, { clusterId: selectedClusterId })
            : clustersPath.replace('/:clusterId?', '');
        history.push({
            pathname: newPath,
            search,
        });
    }, [history, search, selectedClusterId]);

    function toggleAutoUpgrade() {
        // @TODO, wrap this settings change in a confirmation prompt of some sort
        const previousValue = autoUpgradeConfig.enableAutoUpgrade;
        const newConfig = {
            ...autoUpgradeConfig,
            enableAutoUpgrade: !previousValue,
        };

        setAutoUpgradeConfig(newConfig); // optimistically set value before API call

        saveAutoUpgradeConfig(newConfig).catch(() => {
            // reverse the optimistic update of the control in the UI
            const rollbackConfig = {
                ...autoUpgradeConfig,
                enableAutoUpgrade: previousValue,
            };
            setAutoUpgradeConfig(rollbackConfig);

            // also, re-fetch the data from the server, just in case it did update but we didn't get the network response
            fetchConfig();
        });
    }
    const headerText = 'Clusters';
    const subHeaderText = 'Resource list';

    const pageHeader = (
        <PageHeader header={headerText} subHeader={subHeaderText}>
            <div className="flex flex-1 items-center justify-end">
                <URLSearchInput
                    className="w-full"
                    categoryOptions={searchOptions}
                    categories={['CLUSTERS']}
                    placeholder="Add one or more filters"
                    autoFocus={autoFocusSearchInput}
                />
                <div className="flex items-center min-w-64 ml-4">
                    <ToggleSwitch
                        id="enableAutoUpgrade"
                        toggleHandler={toggleAutoUpgrade}
                        label="Automatically upgrade secured clusters"
                        enabled={autoUpgradeConfig.enableAutoUpgrade}
                    />
                </div>
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

const mapStateToProps = createStructuredSelector({
    searchOptions: selectors.getClustersSearchOptions,
});

const mapDispatchToProps = {
    setSearchOptions: clustersActions.setClustersSearchOptions,
};

export default connect(mapStateToProps, mapDispatchToProps)(ClustersPage);
