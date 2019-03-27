import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import dateFns from 'date-fns';
import * as Icon from 'react-feather';

import { actions as graphActions } from 'reducers/network/graph';

class NodesUpdateSection extends Component {
    static propTypes = {
        lastUpdatedEpoch: PropTypes.number,
        lastUpdatedTimestamp: PropTypes.instanceOf(Date),
        networkPolicyGraph: PropTypes.shape({
            epoch: PropTypes.number
        }).isRequired,
        networkNodesUpdate: PropTypes.func.isRequired
    };

    static defaultProps = {
        lastUpdatedTimestamp: null,
        lastUpdatedEpoch: null
    };

    onUpdateGraph = () => {
        this.props.networkNodesUpdate();
    };

    getNodeUpdates = () => {
        const { networkPolicyGraph, lastUpdatedEpoch } = this.props;
        return lastUpdatedEpoch - networkPolicyGraph.epoch;
    };

    renderNodesUpdateButton = () => {
        const nodeUpdatesCount = this.getNodeUpdates();
        if (Number.isNaN(nodeUpdatesCount) || nodeUpdatesCount <= 0) return null;
        return (
            <button
                type="button"
                className="btn-graph-refresh p-1 bg-primary-300 border-2 border-primary-400 hover:bg-primary-200 rounded-sm text-sm text-primary-700 mt-2 w-full font-700"
                onClick={this.onUpdateGraph}
            >
                <Icon.Circle className="h-2 w-2 text-primary-300 border-primary-300" />
                <span className="pl-1">
                    {`${nodeUpdatesCount} update${nodeUpdatesCount === 1 ? '' : 's'} available`}
                </span>
            </button>
        );
    };

    render() {
        if (!this.props.lastUpdatedTimestamp) return null;
        return (
            <div className="absolute pin-t pin-network-update-label-left mt-2 mr-2 p-2 bg-base-100 rounded-sm border-2 border-base-400 text-base-500 text-xs font-700">
                <div className="uppercase">{`Last Updated: ${dateFns.format(
                    this.props.lastUpdatedTimestamp,
                    'hh:mm:ssA'
                )}`}</div>
                {this.renderNodesUpdateButton()}
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    lastUpdatedTimestamp: selectors.getLastUpdatedTimestamp,
    lastUpdatedEpoch: selectors.getNodeUpdatesEpoch,
    networkPolicyGraph: selectors.getNetworkPolicyGraph
});

const mapDispatchToProps = {
    getNodeUpdates: graphActions.setNetworkGraphFilterMode,
    networkNodesUpdate: graphActions.networkNodesUpdate
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(NodesUpdateSection);
