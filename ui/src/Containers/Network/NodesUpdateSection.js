import React, { Component } from 'react';
import PropTypes from 'prop-types';

import dateFns from 'date-fns';
import * as Icon from 'react-feather';

class NodesUpdateSection extends Component {
    static propTypes = {
        getNodeUpdates: PropTypes.func.isRequired,
        onUpdateGraph: PropTypes.func.isRequired,
        lastUpdatedTimestamp: PropTypes.instanceOf(Date)
    };

    static defaultProps = {
        lastUpdatedTimestamp: null
    };

    renderNodesUpdateButton = () => {
        const nodeUpdatesCount = this.props.getNodeUpdates();
        if (Number.isNaN(nodeUpdatesCount) || nodeUpdatesCount <= 0) return null;
        return (
            <button
                type="button"
                className="btn-graph-refresh p-1 bg-primary-300 border-2 border-primary-400 hover:bg-primary-200 rounded-sm text-sm text-primary-700 mt-2 w-full font-700"
                onClick={this.props.onUpdateGraph}
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
            <div className="absolute pin-t pin-network-update-label-left mt-2 mr-2 p-2 bg-base-100 rounded-sm shadow text-base-500 text-sm font-700">
                <div className="uppercase">{`Last Updated: ${dateFns.format(
                    this.props.lastUpdatedTimestamp,
                    'hh:mm:ssA'
                )}`}</div>
                {this.renderNodesUpdateButton()}
            </div>
        );
    }
}

export default NodesUpdateSection;
