import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import dateFns from 'date-fns';
import * as Icon from 'react-feather';

import { actions as graphActions } from 'reducers/network/graph';

const NodesUpdateButton = ({ nodeUpdatesCount, networkNodesUpdate }) => {
    if (Number.isNaN(nodeUpdatesCount) || nodeUpdatesCount <= 0) return null;
    return (
        <button
            type="button"
            className="btn-graph-refresh p-1 bg-primary-300 border-2 border-primary-400 hover:bg-primary-200 rounded-sm text-sm text-primary-700 mt-2 w-full font-700"
            onClick={networkNodesUpdate}
        >
            <Icon.Circle className="h-2 w-2 text-primary-300 border-primary-300" />
            <span className="pl-1">
                {`${nodeUpdatesCount} update${nodeUpdatesCount === 1 ? '' : 's'} available`}
            </span>
        </button>
    );
};

NodesUpdateButton.propTypes = {
    nodeUpdatesCount: PropTypes.number.isRequired,
    networkNodesUpdate: PropTypes.func.isRequired
};

const NodesUpdateSection = ({ networkNodesUpdate, nodeUpdatesCount, lastUpdatedTimestamp }) => {
    if (!lastUpdatedTimestamp) return null;
    return (
        <div className="absolute pin-t pin-network-update-label-left mt-2 mr-2 p-2 bg-base-100 rounded-sm border-2 border-base-400 text-base-500 text-xs font-700">
            <div className="uppercase">{`Last Updated: ${dateFns.format(
                lastUpdatedTimestamp,
                'hh:mm:ssA'
            )}`}</div>
            <NodesUpdateButton
                nodeUpdatesCount={nodeUpdatesCount}
                networkNodesUpdate={networkNodesUpdate}
            />
        </div>
    );
};

NodesUpdateSection.propTypes = {
    lastUpdatedTimestamp: PropTypes.instanceOf(Date),
    nodeUpdatesCount: PropTypes.number.isRequired,
    networkNodesUpdate: PropTypes.func.isRequired
};

NodesUpdateSection.defaultProps = {
    lastUpdatedTimestamp: null
};

const getNodeUpdatesCount = createSelector(
    [selectors.getNetworkPolicyGraph, selectors.getNodeUpdatesEpoch],
    (networkPolicyGraph, lastUpdatedEpoch) => lastUpdatedEpoch - networkPolicyGraph.epoch
);

const mapStateToProps = createStructuredSelector({
    lastUpdatedTimestamp: selectors.getLastUpdatedTimestamp,
    nodeUpdatesCount: getNodeUpdatesCount
});

const mapDispatchToProps = {
    getNodeUpdates: graphActions.setNetworkGraphFilterMode,
    networkNodesUpdate: graphActions.networkNodesUpdate
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(NodesUpdateSection);
