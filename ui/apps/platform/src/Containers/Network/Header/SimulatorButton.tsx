import React, { ReactElement } from 'react';
import { withRouter, useHistory } from 'react-router-dom';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { Button } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import { actions as graphActions } from 'reducers/network/graph';
import { filterModes } from 'constants/networkFilterModes';
import sidepanelStages from '../SidePanel/sidepanelStages';

type SimulatorButtonProps = {
    creatingOrSimulating: boolean;
    openSidePanel: () => void;
    setSidePanelStage: (stage) => void;
    closeSidePanel: () => void;
    setNetworkGraphFilterMode: (mode) => void;
    isDisabled?: boolean;
};

function SimulatorButton({
    creatingOrSimulating,
    openSidePanel,
    setSidePanelStage,
    closeSidePanel,
    setNetworkGraphFilterMode,
    isDisabled = false,
}: SimulatorButtonProps): ReactElement {
    const history = useHistory();

    function toggleSimulation() {
        // @TODO: This isn't very nice. We'll have  to revisit this in the future. But adding this fix to address a customer issue (https://stack-rox.atlassian.net/browse/ROX-3118)
        history.push(`/main/network${history.location.search as string}`);
        if (creatingOrSimulating) {
            closeSidePanel();
        } else {
            openSidePanel();
            setSidePanelStage(sidepanelStages.creator);
            setNetworkGraphFilterMode(filterModes.allowed);
        }
    }

    return (
        <Button
            data-testid={`simulator-button-${creatingOrSimulating ? 'on' : 'off'}`}
            onClick={toggleSimulation}
            isDisabled={isDisabled}
            variant="primary"
        >
            Network Policy Simulator
        </Button>
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
    setNetworkGraphFilterMode: graphActions.setNetworkGraphFilterMode,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(SimulatorButton));
