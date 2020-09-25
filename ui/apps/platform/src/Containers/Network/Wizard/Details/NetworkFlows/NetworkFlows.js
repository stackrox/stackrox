import React, { useState } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as graphActions } from 'reducers/network/graph';

import { knownBackendFlags, isBackendFeatureFlagEnabled } from 'utils/featureFlags';
import { getNetworkFlows } from 'utils/networkGraphUtils';
import { filterModes, filterLabels } from 'constants/networkFilterModes';
import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import NoResultsMessage from 'Components/NoResultsMessage';
import FeatureEnabled from 'Containers/FeatureEnabled/FeatureEnabled';
import useSearchFilteredData from 'hooks/useSearchFilteredData';
import NetworkFlowsSearch from './NetworkFlowsSearch';
import NetworkFlowsTable from './NetworkFlowsTable';
import getNetworkFlowValueByCategory from './networkFlowUtils/getNetworkFlowValueByCategory';

const NetworkFlows = ({
    deploymentEdges,
    filterState,
    onNavigateToDeploymentById,
    featureFlags,
}) => {
    const { networkFlows } = getNetworkFlows(deploymentEdges, filterState);

    const [page, setPage] = useState(0);
    const [searchOptions, setSearchOptions] = useState([]);

    const filteredNetworkFlows = useSearchFilteredData(
        networkFlows,
        searchOptions,
        getNetworkFlowValueByCategory
    );

    const filterStateString = filterState !== filterModes.all ? filterLabels[filterState] : '';

    if (!filteredNetworkFlows.length) {
        return <NoResultsMessage message={`No ${filterStateString} network flows`} />;
    }

    // @TODO: Remove "showPortsAndProtocols" when the feature flag "ROX_NETWORK_GRAPH_PORTS" is defaulted to true
    const showPortsAndProtocols = isBackendFeatureFlagEnabled(
        featureFlags,
        knownBackendFlags.ROX_NETWORK_GRAPH_PORTS,
        false
    );

    const headerComponents = (
        <>
            <FeatureEnabled featureFlag={knownBackendFlags.ROX_NETWORK_FLOWS_SEARCH_FILTER_UI}>
                {({ featureEnabled }) =>
                    featureEnabled && (
                        <div className="flex flex-1">
                            <NetworkFlowsSearch
                                networkFlows={networkFlows}
                                searchOptions={searchOptions}
                                setSearchOptions={setSearchOptions}
                            />
                        </div>
                    )
                }
            </FeatureEnabled>
            <TablePagination
                page={page}
                dataLength={filteredNetworkFlows.length}
                setPage={setPage}
            />
        </>
    );
    const subHeaderText = `${filteredNetworkFlows.length} ${filterStateString} ${pluralize(
        'Flow',
        filteredNetworkFlows.length
    )}`;

    return (
        <div className="w-full h-full">
            <Panel header={subHeaderText} headerComponents={headerComponents} isUpperCase={false}>
                <div className="w-full h-full bg-base-100">
                    <NetworkFlowsTable
                        networkFlows={filteredNetworkFlows}
                        page={page}
                        filterState={filterState}
                        onNavigateToDeploymentById={onNavigateToDeploymentById}
                        showPortsAndProtocols={showPortsAndProtocols}
                    />
                </div>
            </Panel>
        </div>
    );
};

NetworkFlows.propTypes = {
    deploymentEdges: PropTypes.arrayOf(PropTypes.shape({})),
    networkGraphRef: PropTypes.shape({
        setSelectedNode: PropTypes.func,
        getNodeData: PropTypes.func,
        onNodeClick: PropTypes.func,
    }),
    filterState: PropTypes.number.isRequired,
    onNavigateToDeploymentById: PropTypes.func.isRequired,
    featureFlags: PropTypes.arrayOf(PropTypes.shape),
};

NetworkFlows.defaultProps = {
    deploymentEdges: [],
    networkGraphRef: null,
    featureFlags: [],
};

const mapStateToProps = createStructuredSelector({
    networkGraphRef: selectors.getNetworkGraphRef,
    filterState: selectors.getNetworkGraphFilterMode,
    featureFlags: selectors.getFeatureFlags,
});

const mapDispatchToProps = {
    setSelectedNamespace: graphActions.setSelectedNamespace,
};

export default connect(mapStateToProps, mapDispatchToProps)(NetworkFlows);
