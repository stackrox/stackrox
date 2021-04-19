import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';

import sidepanelStages from '../../sidepanelStages';

type GenerateButtonProps = {
    loadActivePolicies: () => void;
    setSidePanelStage: (stage) => void;
};

function GenerateButton({
    setSidePanelStage,
    loadActivePolicies,
}: GenerateButtonProps): ReactElement {
    function onClick() {
        loadActivePolicies();
        setSidePanelStage(sidepanelStages.simulator);
    }

    return (
        <div className="flex items-center ml-2 -mr-2">
            <button
                data-testid="view-active-yaml-button"
                type="button"
                className="mr-4 px-3 py-2 text-xs border-2 border-base-400 bg-base-100 hover:border-primary-400 hover:text-primary-700 font-700 rounded-sm text-center text-base-500 uppercase"
                onClick={onClick}
            >
                View Active YAMLs
            </button>
        </div>
    );
}

const mapDispatchToProps = {
    loadActivePolicies: sidepanelActions.loadActiveNetworkPolicyModification,
    setSidePanelStage: sidepanelActions.setSidePanelStage,
};

export default connect(null, mapDispatchToProps)(GenerateButton);
