import React, { ReactElement } from 'react';
import { withRouter, useHistory } from 'react-router-dom';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import sidepanelStages from '../SidePanel/sidepanelStages';

type SimulatorButtonProps = {
    creatingOrSimulating: boolean;
    openSidePanel: () => void;
    setSidePanelStage: (stage) => void;
    closeSidePanel: () => void;
    isDisabled?: boolean;
};

function SimulatorButton({
    creatingOrSimulating,
    openSidePanel,
    setSidePanelStage,
    closeSidePanel,
    isDisabled = false,
}: SimulatorButtonProps): ReactElement {
    const history = useHistory();

    function toggleSimulation() {
        // @TODO: This isn't very nice. We'll have  to revisit this in the future. But adding this fix to address a customer issue (https://stack-rox.atlassian.net/browse/ROX-3118)
        history.push('/main/network');
        if (creatingOrSimulating) {
            closeSidePanel();
        } else {
            openSidePanel();
            setSidePanelStage(sidepanelStages.creator);
        }
    }

    const className = creatingOrSimulating
        ? 'bg-success-200 border-success-500 hover:border-success-600 hover:text-success-600 text-success-500'
        : 'bg-base-200 hover:border-base-400 hover:text-base-700 border-base-300';
    const iconColor = creatingOrSimulating ? '#53c6a9' : '#d2d5ed';
    return (
        <button
            type="button"
            data-testid={`simulator-button-${creatingOrSimulating ? 'on' : 'off'}`}
            className={`flex items-center flex-shrink-0 border-2 rounded-sm text-sm pl-2 pr-2 h-10 ${className}`}
            onClick={toggleSimulation}
            disabled={isDisabled}
        >
            <Icon.Circle className="h-2 w-2" fill={iconColor} stroke={iconColor} />
            <span className="pl-2">Network Policy Simulator</span>
        </button>
    );
}

const getCreatingOrSimulating = createSelector(
    [selectors.getSidePanelOpen, selectors.getSidePanelStage],
    (sidePanelOpen: boolean, sidePanelStage: string): boolean =>
        sidePanelOpen &&
        (sidePanelStage === sidepanelStages.simulator || sidePanelStage === sidepanelStages.creator)
);

const mapStateToProps = createStructuredSelector({
    creatingOrSimulating: getCreatingOrSimulating,
});

const mapDispatchToProps = {
    openSidePanel: pageActions.openSidePanel,
    closeSidePanel: pageActions.closeSidePanel,
    setSidePanelStage: sidepanelActions.setSidePanelStage,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(SimulatorButton));
