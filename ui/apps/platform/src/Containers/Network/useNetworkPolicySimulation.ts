import { useSelector, useDispatch } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import { actions as dialogueActions } from 'reducers/network/dialogue';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as pageActions } from 'reducers/network/page';
import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';
import wizardStages from 'Containers/Network/Wizard/wizardStages';

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
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,
    modificationState: getModificationState,
});

const useNetworkPolicySimulation = (): NetworkPolicySimulationResult => {
    const { wizardOpen, wizardStage, modificationState } = useSelector(structuredSelector);
    const dispatch = useDispatch();
    const stopNetworkSimulation = () => {
        dispatch(pageActions.closeNetworkWizard());
        dispatch(dialogueActions.setNetworkDialogueStage(dialogueStages.closed));
        dispatch(wizardActions.setNetworkPolicyModification(null));
    };

    const isNetworkSimulationOn =
        wizardOpen &&
        (wizardStage === wizardStages.simulator || wizardStage === wizardStages.creator);

    return {
        isNetworkSimulationOn,
        isNetworkSimulationError: modificationState === 'ERROR',
        stopNetworkSimulation,
    };
};

export default useNetworkPolicySimulation;
