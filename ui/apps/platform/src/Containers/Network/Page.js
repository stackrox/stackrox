import React, { useEffect, useState } from 'react';
import { connect, useDispatch, useSelector } from 'react-redux';
import { useRouteMatch } from 'react-router-dom';
import { createStructuredSelector } from 'reselect';
import { Alert } from '@patternfly/react-core';

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
import { getErrorMessageFromServerResponse } from 'utils/networkGraphUtils';
import Header from './Header/Header';
import NoSelectedNamespace from './NoSelectedNamespace';
import GraphLoadErrorState from './GraphLoadErrorState';

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

function NetworkPage({
    getNetworkFlowGraphState,
    serverErrorMessage,
    closeSidePanel,
    setDialogueStage,
    setNetworkModification,
}) {
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
        if (!isInitialRender) {
            return;
        }
        setIsInitialRender(false);
        if (!deploymentId) {
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

    const hasNoSelectedNamespace = selectedNamespaceFilters.length === 0;
    const hasGraphLoadError = getNetworkFlowGraphState === 'ERROR';

    let content;
    if (hasNoSelectedNamespace) {
        content = <NoSelectedNamespace clusterName={clusterName} />;
    } else if (hasGraphLoadError) {
        const userMessage = getErrorMessageFromServerResponse(serverErrorMessage);
        content = <GraphLoadErrorState error={serverErrorMessage} userMessage={userMessage} />;
    } else {
        content = <GraphFrame />;
    }

    return (
        <>
            <Alert
                isInline
                variant="warning"
                title={
                    <p>
                        Version 1.0 of Network Graph is being deprecated soon. Please switch to the
                        new 2.0 version for improved functionality and a better user experience.
                        Contact our support team for assistance
                    </p>
                }
            />
            <Header
                isGraphDisabled={hasNoSelectedNamespace || hasGraphLoadError}
                isSimulationOn={isSimulationOn}
            />
            <section className="flex flex-1 h-full w-full">
                <div className="flex flex-1 flex-col w-full overflow-hidden">
                    <div className="flex flex-1 flex-col relative">{content}</div>
                </div>
                <Dialogue />
            </section>
        </>
    );
}

const mapStateToProps = createStructuredSelector({
    getNetworkFlowGraphState: selectors.getNetworkFlowGraphState,
    serverErrorMessage: selectors.getNetworkFlowErrorMessage,
});

const mapDispatchToProps = {
    closeSidePanel: pageActions.closeSidePanel,
    setNetworkModification: sidepanelActions.setNetworkPolicyModification,
    setDialogueStage: dialogueActions.setNetworkDialogueStage,
};

export default connect(mapStateToProps, mapDispatchToProps)(NetworkPage);
