import { useSelector, useDispatch } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import { actions as dialogueActions } from 'reducers/network/dialogue';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import { actions as pageActions } from 'reducers/network/page';
import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';
import sidepanelStages from 'Containers/Network/SidePanel/sidepanelStages';

type ModificationState = 'SUCCESS' | 'REQUEST' | 'ERROR';
type NetworkPolicySimulationResult = {
    isNetworkSimulationOn: boolean;
    isNetworkSimulationError: boolean;
    stopNetworkSimulation: () => void;
};

const getModificationState = createSelector(
    [selectors.getNetworkPolicyModification, selectors.getNetworkPolicyModificationState],
    (modification, modificationState: ModificationState): ModificationState | 'INITIAL' => {
        if (!modification) {
            return 'INITIAL';
        }
        return modificationState;
    }
);

const structuredSelector = createStructuredSelector({
    sidePanelOpen: selectors.getSidePanelOpen,
    sidePanelStage: selectors.getSidePanelStage,
    modificationState: getModificationState,
});

const useNetworkPolicySimulation = (): NetworkPolicySimulationResult => {
    const { sidePanelOpen, sidePanelStage, modificationState } = useSelector(structuredSelector);
    const dispatch = useDispatch();
    const stopNetworkSimulation = () => {
        dispatch(pageActions.closeSidePanel());
        dispatch(dialogueActions.setNetworkDialogueStage(dialogueStages.closed));
        dispatch(sidepanelActions.setNetworkPolicyModification(null));
    };

    const isNetworkSimulationOn =
        sidePanelOpen &&
        (sidePanelStage === sidepanelStages.simulator ||
            sidePanelStage === sidepanelStages.creator);

    return {
        isNetworkSimulationOn,
        isNetworkSimulationError: modificationState === 'ERROR',
        stopNetworkSimulation,
    };
};

export default useNetworkPolicySimulation;
