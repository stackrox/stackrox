import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import dateFns from 'date-fns';
import * as Icon from 'react-feather';

import { selectors } from 'reducers';
import { actions as graphActions } from 'reducers/network/graph';

type NodesUpdateButtonProps = {
    nodeUpdatesCount: number;
    updateNetworkNodes: () => void;
};

function NodesUpdateButton({
    nodeUpdatesCount,
    updateNetworkNodes,
}: NodesUpdateButtonProps): ReactElement {
    return (
        <button
            type="button"
            className="btn-graph-refresh p-1 bg-primary-300 border-2 border-primary-400 hover:bg-primary-200 rounded-sm text-sm text-primary-700 mt-2 w-full font-700 flex items-center"
            onClick={updateNetworkNodes}
        >
            <Icon.Circle className="h-2 w-2 text-primary-300 border-primary-300" />
            <span className="pl-1">
                {`${nodeUpdatesCount} update${nodeUpdatesCount === 1 ? '' : 's'} available`}
            </span>
        </button>
    );
}

type NodeUpdateSectionProps = {
    lastUpdatedTimestamp: Date;
    nodeUpdatesCount: number;
    updateNetworkNodes: () => void;
};

const NodesUpdateSection = ({
    updateNetworkNodes,
    nodeUpdatesCount,
    lastUpdatedTimestamp,
}: NodeUpdateSectionProps) => {
    return (
        <div
            data-testid="nodes-update-section"
            data-test-updated={dateFns.format(lastUpdatedTimestamp, 'hhmmssSSSS')}
            className="absolute top-0 pin-network-update-label-left mt-2 mr-2 p-2 bg-base-100 rounded-sm border-2 border-base-400 text-base-500 text-xs font-700"
        >
            <div className="uppercase">{`Last Updated: ${dateFns.format(
                lastUpdatedTimestamp,
                'hh:mm:ssA'
            )}`}</div>
            {nodeUpdatesCount > 0 && (
                <NodesUpdateButton
                    nodeUpdatesCount={nodeUpdatesCount}
                    updateNetworkNodes={updateNetworkNodes}
                />
            )}
        </div>
    );
};

const getNodeUpdatesCount = createSelector(
    [selectors.getNetworkPolicyGraph, selectors.getNodeUpdatesEpoch],
    (networkPolicyGraph, lastUpdatedEpoch: number) => lastUpdatedEpoch - networkPolicyGraph.epoch
);

const mapStateToProps = createStructuredSelector({
    nodeUpdatesCount: getNodeUpdatesCount,
});

const mapDispatchToProps = {
    getNodeUpdates: graphActions.setNetworkGraphFilterMode,
    updateNetworkNodes: graphActions.updateNetworkNodes,
};

export default connect(mapStateToProps, mapDispatchToProps)(NodesUpdateSection);
