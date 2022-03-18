import React, { ReactElement } from 'react';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Button } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import sidepanelStages from '../SidePanel/sidepanelStages';

type CIDRFormButtonProps = {
    sidePanelOpen: boolean;
    sidePanelStage: string;
    openSidePanel: () => void;
    closeSidePanel: () => void;
    setSidePanelStage: (stage) => void;
    isDisabled: boolean;
};

function CIDRFormButton({
    sidePanelOpen,
    sidePanelStage,
    openSidePanel,
    closeSidePanel,
    setSidePanelStage,
    isDisabled,
}: CIDRFormButtonProps): ReactElement {
    function toggleForm() {
        if (sidePanelOpen && sidePanelStage === sidepanelStages.cidrForm) {
            closeSidePanel();
        } else {
            if (!sidePanelOpen) {
                openSidePanel();
            }
            setSidePanelStage(sidepanelStages.cidrForm);
        }
    }

    return (
        <Button variant="secondary" onClick={toggleForm} isDisabled={isDisabled}>
            CIDR Blocks
        </Button>
    );
}

const mapStateToProps = createStructuredSelector({
    sidePanelOpen: selectors.getSidePanelOpen,
    sidePanelStage: selectors.getSidePanelStage,
});

const mapDispatchToProps = {
    openSidePanel: pageActions.openSidePanel,
    closeSidePanel: pageActions.closeSidePanel,
    setSidePanelStage: sidepanelActions.setSidePanelStage,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(CIDRFormButton));
