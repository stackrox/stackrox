import React, { useState } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as graphActions } from 'reducers/network/graph';

import { getNetworkFlows } from 'utils/networkUtils/getNetworkFlows';
import { filterModes, filterLabels } from 'constants/networkFilterModes';
import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import NoResultsMessage from 'Components/NoResultsMessage';
import useSearchFilteredData from 'hooks/useSearchFilteredData';
import NetworkFlowsSearch from './NetworkFlowsSearch';
import NetworkFlowsTable from './NetworkFlowsTable';
import getNetworkFlowValueByCategory from './networkFlowUtils/getNetworkFlowValueByCategory';

const NetworkFlows = ({ edges, filterState, onNavigateToDeploymentById }) => {
    const { networkFlows } = getNetworkFlows(edges, filterState);

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

    const headerComponents = (
        <>
            <div className="flex flex-1">
                <NetworkFlowsSearch
                    networkFlows={networkFlows}
                    searchOptions={searchOptions}
                    setSearchOptions={setSearchOptions}
                />
            </div>
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
                    />
                </div>
            </Panel>
        </div>
    );
};

NetworkFlows.propTypes = {
    edges: PropTypes.arrayOf(PropTypes.shape({})),
    networkGraphRef: PropTypes.shape({
        setSelectedNode: PropTypes.func,
        getNodeData: PropTypes.func,
        onNodeClick: PropTypes.func,
    }),
    filterState: PropTypes.number.isRequired,
    onNavigateToDeploymentById: PropTypes.func.isRequired,
};

NetworkFlows.defaultProps = {
    edges: [],
    networkGraphRef: null,
};

const mapStateToProps = createStructuredSelector({
    networkGraphRef: selectors.getNetworkGraphRef,
    filterState: selectors.getNetworkGraphFilterMode,
});

const mapDispatchToProps = {
    setSelectedNamespace: graphActions.setSelectedNamespace,
};

export default connect(mapStateToProps, mapDispatchToProps)(NetworkFlows);
