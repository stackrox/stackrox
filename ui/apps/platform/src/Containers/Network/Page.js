import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import { actions as dialogueActions } from 'reducers/network/dialogue';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as pageActions } from 'reducers/network/page';
import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';
import useNetworkPolicySimulation from 'Containers/Network/useNetworkPolicySimulation';

import SimulationFrame from 'Components/SimulationFrame';
import Dialogue from 'Containers/Network/Dialogue';
import Graph from 'Containers/Network/Graph/Graph';
import Header from 'Containers/Network/Header/Header';
import Wizard from 'Containers/Network/Wizard/Wizard';

function NetworkPage({ closeWizard, setDialogueStage, setNetworkModification }) {
    const {
        isNetworkSimulationOn,
        isNetworkSimulationError,
        stopNetworkSimulation,
    } = useNetworkPolicySimulation();

    useEffect(() => {
        return () => {
            closeWizard();
            setDialogueStage(dialogueStages.closed);
            setNetworkModification(null);
        };
    }, [closeWizard, setDialogueStage, setNetworkModification]);

    const content = isNetworkSimulationOn ? (
        <SimulationFrame isError={isNetworkSimulationError} onStop={stopNetworkSimulation}>
            <Graph />
            <Wizard />
        </SimulationFrame>
    ) : (
        <div className="flex flex-1 relative">
            <Graph />
            <Wizard />
        </div>
    );

    return (
        <section className="flex flex-1 h-full w-full">
            <div className="flex flex-1 flex-col w-full overflow-hidden">
                <div className="flex border-b border-base-400">
                    <Header />
                </div>
                {content}
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
