import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import useLocalStorage from 'hooks/useLocalStorage';
import { actions as dialogueActions } from 'reducers/network/dialogue';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import { actions as pageActions } from 'reducers/network/page';
import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';
import useNetworkPolicySimulation from 'Containers/Network/useNetworkPolicySimulation';
import useNetworkBaselineSimulation from 'Containers/Network/useNetworkBaselineSimulation';
import useFetchBaselineComparisons from 'Containers/Network/useFetchBaselineComparisons';

import SimulationFrame from 'Components/SimulationFrame';
import Dialogue from 'Containers/Network/Dialogue';
import Graph from 'Containers/Network/Graph/Graph';
import SidePanel from 'Containers/Network/SidePanel/SidePanel';
import Header from './Header/Header';

function GraphFrame() {
    const [showNamespaceFlows, setShowNamespaceFlows] = useLocalStorage(
        'showNamespaceFlows',
        'show'
    );
    const { isNetworkSimulationOn, isNetworkSimulationError, stopNetworkSimulation } =
        useNetworkPolicySimulation();
    const { isBaselineSimulationOn, stopBaselineSimulation } = useNetworkBaselineSimulation();
    const { simulatedBaselines } = useFetchBaselineComparisons();

    const isSimulationOn = isNetworkSimulationOn || isBaselineSimulationOn;
    let onStop;
    if (isNetworkSimulationOn) {
        onStop = stopNetworkSimulation;
    }
    if (isBaselineSimulationOn) {
        onStop = stopBaselineSimulation;
    }
    const isError = isNetworkSimulationOn && isNetworkSimulationError;

    function handleNamespaceFlowsToggle(mode) {
        setShowNamespaceFlows(mode);
    }

    return isSimulationOn ? (
        <SimulationFrame isError={isError} onStop={onStop}>
            <div className="flex flex-1 relative">
                <Graph
                    isSimulationOn
                    showNamespaceFlows={showNamespaceFlows}
                    setShowNamespaceFlows={handleNamespaceFlowsToggle}
                    simulatedBaselines={simulatedBaselines}
                />
                <SidePanel />
            </div>
        </SimulationFrame>
    ) : (
        <div className="flex flex-1 relative">
            <Graph
                showNamespaceFlows={showNamespaceFlows}
                setShowNamespaceFlows={handleNamespaceFlowsToggle}
            />
            <SidePanel />
        </div>
    );
}

function NetworkPage({ closeSidePanel, setDialogueStage, setNetworkModification }) {
    const { isNetworkSimulationOn } = useNetworkPolicySimulation();
    const { isBaselineSimulationOn } = useNetworkBaselineSimulation();
    const isSimulationOn = isNetworkSimulationOn || isBaselineSimulationOn;

    // when this component unmounts, then close the side panel and exit network policy simulation
    useEffect(() => {
        return () => {
            closeSidePanel();
            setDialogueStage(dialogueStages.closed);
            setNetworkModification(null);
        };
    }, [closeSidePanel, setDialogueStage, setNetworkModification]);

    return (
        <>
            <Header isSimulationOn={isSimulationOn} />
            <section className="flex flex-1 h-full w-full">
                <div className="flex flex-1 flex-col w-full overflow-hidden">
                    <div className="flex flex-1 flex-col relative">
                        <GraphFrame />
                    </div>
                </div>
                <Dialogue />
            </section>
        </>
    );
}

const mapDispatchToProps = {
    closeSidePanel: pageActions.closeSidePanel,
    setNetworkModification: sidepanelActions.setNetworkPolicyModification,
    setDialogueStage: dialogueActions.setNetworkDialogueStage,
};

export default connect(null, mapDispatchToProps)(NetworkPage);
