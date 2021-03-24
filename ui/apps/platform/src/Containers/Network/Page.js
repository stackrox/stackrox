import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import { actions as dialogueActions } from 'reducers/network/dialogue';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as pageActions } from 'reducers/network/page';
import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';
import useNetworkPolicySimulation from 'Containers/Network/useNetworkPolicySimulation';
import {
    BaselineSimulationProvider,
    useNetworkBaselineSimulation,
} from 'Containers/Network/baselineSimulationContext';

import SimulationFrame from 'Components/SimulationFrame';
import Dialogue from 'Containers/Network/Dialogue';
import Graph from 'Containers/Network/Graph/Graph';
import Header from 'Containers/Network/Header/Header';
import Wizard from 'Containers/Network/Wizard/Wizard';

function NetworkPageContent() {
    const {
        isNetworkSimulationOn,
        isNetworkSimulationError,
        stopNetworkSimulation,
    } = useNetworkPolicySimulation();
    const { isBaselineSimulationOn, stopBaselineSimulation } = useNetworkBaselineSimulation();

    const isSimulationOn = isNetworkSimulationOn || isBaselineSimulationOn;
    let onStop;
    if (isNetworkSimulationOn) {
        onStop = stopNetworkSimulation;
    }
    if (isBaselineSimulationOn) {
        onStop = stopBaselineSimulation;
    }
    const isError = isNetworkSimulationOn && isNetworkSimulationError;

    return (
        <div className="flex flex-1 flex-col relative">
            <div className="flex border-b border-base-400">
                <Header isDisabled={isSimulationOn} />
            </div>
            {isSimulationOn ? (
                <SimulationFrame isError={isError} onStop={onStop}>
                    <Graph />
                    <Wizard />
                </SimulationFrame>
            ) : (
                <div className="flex flex-1 relative">
                    <Graph />
                    <Wizard />
                </div>
            )}
        </div>
    );
}

function NetworkPage({ closeWizard, setDialogueStage, setNetworkModification }) {
    // when this component unmounts, then close the side panel and exit network policy simulation
    useEffect(() => {
        return () => {
            closeWizard();
            setDialogueStage(dialogueStages.closed);
            setNetworkModification(null);
        };
    }, [closeWizard, setDialogueStage, setNetworkModification]);

    return (
        <section className="flex flex-1 h-full w-full">
            <div className="flex flex-1 flex-col w-full overflow-hidden">
                <BaselineSimulationProvider>
                    <NetworkPageContent />
                </BaselineSimulationProvider>
            </div>
            <Dialogue />
        </section>
    );
}

const mapDispatchToProps = {
    closeWizard: pageActions.closeNetworkWizard,
    setNetworkModification: wizardActions.setNetworkPolicyModification,
    setDialogueStage: dialogueActions.setNetworkDialogueStage,
};

export default connect(null, mapDispatchToProps)(NetworkPage);
