import React, { useEffect, useState } from 'react';
import { connect, useDispatch, useSelector } from 'react-redux';
import { useRouteMatch } from 'react-router-dom';
import { createSelector, createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import useLocalStorage from 'hooks/useLocalStorage';
import { actions as dialogueActions } from 'reducers/network/dialogue';
import { actions as graphActions } from 'reducers/network/graph';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import { actions as pageActions } from 'reducers/network/page';
import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';
import useNetworkPolicySimulation from 'Containers/Network/useNetworkPolicySimulation';
import useNetworkBaselineSimulation from 'Containers/Network/useNetworkBaselineSimulation';
import useFetchBaselineComparisons from 'Containers/Network/useFetchBaselineComparisons';
import Dialogue from 'Containers/Network/Dialogue';
import Graph from 'Containers/Network/Graph/Graph';
import SidePanel from 'Containers/Network/SidePanel/SidePanel';
import SimulationFrame from 'Components/SimulationFrame';
import { fetchDeployment } from 'services/DeploymentsService';
import Header from './Header/Header';
import NoSelectedNamespace from './NoSelectedNamespace';

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

const networkPageContentSelector = createStructuredSelector({
    clusters: selectors.getClusters,
    selectedClusterId: selectors.getSelectedNetworkClusterId,
    selectedNamespaceFilters: selectors.getSelectedNamespaceFilters,
});

function NetworkPage({ closeSidePanel, setDialogueStage, setNetworkModification }) {
    const { isNetworkSimulationOn } = useNetworkPolicySimulation();
    const { isBaselineSimulationOn } = useNetworkBaselineSimulation();
    const isSimulationOn = isNetworkSimulationOn || isBaselineSimulationOn;
    const [isInitialRender, setIsInitialRender] = useState(true);

    const {
        params: { deploymentId },
    } = useRouteMatch();
    const { clusters, selectedClusterId, selectedNamespaceFilters } = useSelector(
        networkPageContentSelector
    );
    const dispatch = useDispatch();

    useEffect(() => {
        if (!deploymentId || !isInitialRender) {
            return;
        }
        // If the page is visited with a deployment id, we need to enable that deployment's
        // namespace filter and switch to the correct cluster
        fetchDeployment(deploymentId).then(({ clusterId, namespace }) => {
            if (clusterId !== selectedClusterId) {
                dispatch(graphActions.selectNetworkClusterId(clusterId));
                dispatch(graphActions.setSelectedNamespaceFilters([namespace]));
            } else if (!selectedNamespaceFilters.includes(namespace)) {
                const newFilters = [...selectedNamespaceFilters, namespace];
                dispatch(graphActions.setSelectedNamespaceFilters(newFilters));
            }
        });
        setIsInitialRender(false);
    }, [dispatch, deploymentId, selectedClusterId, selectedNamespaceFilters, isInitialRender]);

    const clusterName = clusters.find((c) => c.id === selectedClusterId)?.name;

    // when this component unmounts, then close the side panel and exit network policy simulation
    useEffect(() => {
        return () => {
            closeSidePanel();
            setDialogueStage(dialogueStages.closed);
            setNetworkModification(null);
        };
    }, [closeSidePanel, setDialogueStage, setNetworkModification]);

    const isGraphDisabled = selectedNamespaceFilters.length === 0;

    return (
        <>
            <Header isGraphDisabled={isGraphDisabled} isSimulationOn={isSimulationOn} />
            <section className="flex flex-1 h-full w-full">
                <div className="flex flex-1 flex-col w-full overflow-hidden">
                    <div className="flex flex-1 flex-col relative">
                        {isGraphDisabled ? (
                            <NoSelectedNamespace clusterName={clusterName} />
                        ) : (
                            <GraphFrame />
                        )}
                    </div>
                </div>
                <Dialogue />
            </section>
        </>
    );
}

const isViewFiltered = createSelector(
    [selectors.getNetworkSearchOptions],
    (searchOptions) => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    isViewFiltered,
});

const mapDispatchToProps = {
    closeSidePanel: pageActions.closeSidePanel,
    setNetworkModification: sidepanelActions.setNetworkPolicyModification,
    setDialogueStage: dialogueActions.setNetworkDialogueStage,
};

export default connect(mapStateToProps, mapDispatchToProps)(NetworkPage);
