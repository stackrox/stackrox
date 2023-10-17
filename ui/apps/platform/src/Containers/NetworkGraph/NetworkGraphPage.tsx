import React, { useEffect, useState } from 'react';
import {
    Bullseye,
    Button,
    Divider,
    Drawer,
    DrawerContent,
    DrawerContentBody,
    DrawerPanelContent,
    PageSection,
    Spinner,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';
import { useHistory } from 'react-router-dom';
import useDeepCompareEffect from 'use-deep-compare-effect';

import { networkBasePath, nonGlobalResourceNamesForNetworkGraph } from 'routePaths';
import { timeWindows } from 'constants/timeWindows';
import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import useFetchDeploymentCount from 'hooks/useFetchDeploymentCount';
import usePermissions from 'hooks/usePermissions';
import useURLSearch from 'hooks/useURLSearch';
import { fetchNetworkFlowGraph, fetchNodeUpdates } from 'services/NetworkService';
import queryService from 'utils/queryService';
import timeWindowToDate from 'utils/timeWindows';
import { isCompleteSearchFilter } from 'utils/searchUtils';

import PageTitle from 'Components/PageTitle';
import useInterval from 'hooks/useInterval';
import useURLParameter from 'hooks/useURLParameter';
import NetworkGraphContainer, { Models } from './NetworkGraphContainer';
import EmptyUnscopedState from './components/EmptyUnscopedState';
import NetworkBreadcrumbs from './components/NetworkBreadcrumbs';
import NodeUpdateSection from './components/NodeUpdateSection';
import NetworkSearch from './components/NetworkSearch';
import SimulateNetworkPolicyButton from './simulation/SimulateNetworkPolicyButton';
import EdgeStateSelect, { EdgeState } from './components/EdgeStateSelect';
import DisplayOptionsSelect, { DisplayOption } from './components/DisplayOptionsSelect';
import TimeWindowSelector from './components/TimeWindowSelector';
import { useScopeHierarchy } from './hooks/useScopeHierarchy';
import useNetworkPolicySimulator from './hooks/useNetworkPolicySimulator';
import {
    transformPolicyData,
    transformActiveData,
    createExtraneousFlowsModel,
    graphModel,
} from './utils/modelUtils';
import getSimulation from './utils/getSimulation';
import { getSearchFilterFromScopeHierarchy } from './utils/simulatorUtils';
import CIDRFormModal from './components/CIDRFormModal';
import NetworkPolicySimulatorSidePanel, {
    clearSimulationQuery,
} from './simulation/NetworkPolicySimulatorSidePanel';

import './NetworkGraphPage.css';

const emptyModel = {
    graph: graphModel,
    nodes: [],
    edges: [],
};

// TODO: get real includePorts flag from user input
const includePorts = true;

// for MVP, always show Orchestrator Components
const ALWAYS_SHOW_ORCHESTRATOR_COMPONENTS = true;

// This is a query param used to add policy data in the response for the network graph data
const INCLUDE_POLICIES = true;

function NetworkGraphPage() {
    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForBlocks =
        hasReadAccess('Administration') && hasReadWriteAccess('NetworkGraph');
    const hasReadAccessForGenerator =
        hasReadAccess('Integration') && hasReadAccess('NetworkPolicy');

    const history = useHistory();
    const [edgeState, setEdgeState] = useState<EdgeState>('active');
    const [displayOptions, setDisplayOptions] = useState<DisplayOption[]>([
        'policyStatusBadge',
        'externalBadge',
        'edgeLabel',
        'selectionIndicator',
        'objectTypeLabel',
    ]);
    const [models, setModels] = useState<Models>({
        activeModel: emptyModel,
        extraneousModel: emptyModel,
    });
    const [previouslySelectedCluster, setPreviouslySelectedCluster] = useState<string | undefined>(
        undefined
    );

    const [isLoading, setIsLoading] = useState(false);
    const [timeWindow, setTimeWindow] = useState<(typeof timeWindows)[number]>(timeWindows[0]);
    const [lastUpdatedTime, setLastUpdatedTime] = useState<string>('');
    const [isCIDRBlockFormOpen, setIsCIDRBlockFormOpen] = useState(false);

    const { searchFilter, setSearchFilter } = useURLSearch();
    const [simulationQueryValue] = useURLParameter('simulation', undefined);
    const simulation = getSimulation(simulationQueryValue);

    const { clusters } = useFetchClustersForPermissions(nonGlobalResourceNamesForNetworkGraph);
    const scopeHierarchy = useScopeHierarchy(clusters);
    const {
        cluster: clusterFromUrl,
        namespaces: namespacesFromUrl,
        deployments: deploymentsFromUrl,
        remainingQuery,
    } = scopeHierarchy;

    if (clusterFromUrl.name !== previouslySelectedCluster) {
        setModels({
            activeModel: emptyModel,
            extraneousModel: emptyModel,
        });
        setPreviouslySelectedCluster(clusterFromUrl.name);
    }

    const hasClusterNamespaceSelected = clusterFromUrl.name !== '' && namespacesFromUrl.length > 0;

    // if no cluster is selected, and there is only one cluster available, automatically select it
    if (clusters.length === 1 && clusterFromUrl.name === '') {
        const modifiedSearchObject = { ...searchFilter };
        modifiedSearchObject.Cluster = clusters[0].name;
        delete modifiedSearchObject.Namespace;
        delete modifiedSearchObject.Deployment;
        setSearchFilter(modifiedSearchObject);
    }

    const selectedClusterId = clusterFromUrl.id;

    const { deploymentCount } = useFetchDeploymentCount(
        getSearchFilterFromScopeHierarchy(scopeHierarchy)
    );

    const [prevEpochCount, setPrevEpochCount] = useState(0);
    const [currentEpochCount, setCurrentEpochCount] = useState(0);

    const nodeUpdatesCount = currentEpochCount - prevEpochCount;

    // We will update the poll epoch after 30 seconds to update the node count for a cluster
    useInterval(() => {
        if (selectedClusterId && namespacesFromUrl.length > 0) {
            fetchNodeUpdates(selectedClusterId)
                .then((result) => {
                    setCurrentEpochCount(result?.response?.epoch || 0);
                })
                .catch(() => {
                    // failure to update the node count is not critical
                });
        }
    }, 30000);

    function updateNetworkNodes() {
        // check that user is finished adding a complete filter
        const isQueryFilterComplete = isCompleteSearchFilter(remainingQuery);

        // only refresh the graph data from the API if both a cluster and at least one namespace are selected
        // and the selected scope has at least one deployment
        const isClusterNamespaceSelected =
            clusterFromUrl.name && namespacesFromUrl.length > 0 && deploymentCount;

        if (isQueryFilterComplete && selectedClusterId && isClusterNamespaceSelected) {
            setIsLoading(true);

            const queryToUse = queryService.objectToWhereClause(remainingQuery);
            const timestampToUse = timeWindowToDate(timeWindow);

            Promise.all([
                // fetch the network graph data used for the active graph
                fetchNetworkFlowGraph(
                    selectedClusterId,
                    namespacesFromUrl,
                    deploymentsFromUrl,
                    queryToUse,
                    timestampToUse || undefined,
                    includePorts,
                    ALWAYS_SHOW_ORCHESTRATOR_COMPONENTS
                ),
                // fetch the network graph data, including policies, for the inactive graph
                fetchNetworkFlowGraph(
                    selectedClusterId,
                    namespacesFromUrl,
                    deploymentsFromUrl,
                    queryToUse,
                    undefined,
                    includePorts,
                    ALWAYS_SHOW_ORCHESTRATOR_COMPONENTS,
                    INCLUDE_POLICIES
                ),
            ])
                .then((values) => {
                    // get policy nodes from policy graph API response
                    const { nodes: policyNodes } = values[1].response;
                    // transform policy data to DataModel
                    const { policyDataModel, policyNodeMap } = transformPolicyData(policyNodes);
                    // get active nodes from network flow graph API response
                    const { nodes: activeNodes } = values[0].response;
                    // transform active data to DataModel
                    const { activeDataModel, activeEdgeMap, activeNodeMap } = transformActiveData(
                        activeNodes,
                        policyNodeMap,
                        namespacesFromUrl
                    );

                    // create extraneous flows graph
                    const extraneousFlowsDataModel = createExtraneousFlowsModel(
                        policyDataModel,
                        activeNodeMap,
                        activeEdgeMap,
                        namespacesFromUrl
                    );

                    const newUpdatedTimestamp = new Date();
                    // show only hours and minutes, use options with the default locale - use an empty array
                    const lastUpdatedDisplayTime = newUpdatedTimestamp.toLocaleTimeString([], {
                        hour: 'numeric',
                        minute: '2-digit',
                    });
                    setLastUpdatedTime(lastUpdatedDisplayTime);

                    // Set the epoch to the most recent value from the server, since the state should now be up to date
                    // with that value at worst.
                    setPrevEpochCount(currentEpochCount);

                    setModels({
                        activeModel: activeDataModel,
                        extraneousModel: extraneousFlowsDataModel,
                    });
                })
                .catch(() => {
                    // TODO
                })
                .finally(() => setIsLoading(false));
        }
    }

    // Epoch counts are tracked separately for each cluster, so reset them when the cluster changes.
    useEffect(() => {
        setPrevEpochCount(0);
        setCurrentEpochCount(0);
    }, [selectedClusterId]);

    // TODO - This ignores some dependencies that would normally be included in the dep array of a regular `useEffect`.
    // We can probably remove the effect and rely on callbacks when setting the parameters instead.
    useDeepCompareEffect(() => {
        updateNetworkNodes();
    }, [
        clusterFromUrl,
        namespacesFromUrl,
        deploymentsFromUrl,
        remainingQuery,
        timeWindow,
        deploymentCount,
    ]);

    const { simulator, setNetworkPolicyModification } = useNetworkPolicySimulator({
        simulation,
        scopeHierarchy,
    });

    function toggleCIDRBlockForm() {
        setIsCIDRBlockFormOpen(!isCIDRBlockFormOpen);
    }

    function closeSimulatorSidebar() {
        const queryString = clearSimulationQuery(history.location.search);
        history.push(`${networkBasePath}${queryString}`);
    }

    return (
        <>
            <PageTitle title="Network Graph" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Toolbar
                    className="network-graph-selector-bar"
                    data-testid="network-graph-selector-bar"
                >
                    <ToolbarContent>
                        <ToolbarGroup variant="filter-group">
                            <Title headingLevel="h1" className="pf-u-screen-reader">
                                Network Graph
                            </Title>
                            <NetworkBreadcrumbs
                                clusters={clusters}
                                selectedCluster={clusterFromUrl}
                                selectedNamespaces={namespacesFromUrl}
                                selectedDeployments={deploymentsFromUrl}
                            />
                        </ToolbarGroup>
                        {(hasWriteAccessForBlocks || hasReadAccessForGenerator) && (
                            <ToolbarGroup
                                variant="button-group"
                                alignment={{ default: 'alignRight' }}
                                spaceItems={{ default: 'spaceItemsMd' }}
                            >
                                {hasWriteAccessForBlocks && (
                                    <ToolbarItem>
                                        <Button
                                            variant="secondary"
                                            onClick={toggleCIDRBlockForm}
                                            isDisabled={!selectedClusterId}
                                        >
                                            Manage CIDR blocks
                                        </Button>
                                    </ToolbarItem>
                                )}
                                {hasReadAccessForGenerator && (
                                    <ToolbarItem>
                                        <SimulateNetworkPolicyButton
                                            simulation={simulation}
                                            isDisabled={scopeHierarchy.cluster.id === ''}
                                        />
                                    </ToolbarItem>
                                )}
                            </ToolbarGroup>
                        )}
                    </ToolbarContent>
                </Toolbar>
            </PageSection>
            <Divider component="div" />
            {hasClusterNamespaceSelected && (
                <>
                    <PageSection variant="light" padding={{ default: 'noPadding' }}>
                        <Toolbar data-testid="network-graph-toolbar">
                            <ToolbarContent>
                                <ToolbarGroup variant="filter-group">
                                    <ToolbarItem>
                                        <EdgeStateSelect
                                            edgeState={edgeState}
                                            setEdgeState={setEdgeState}
                                            isDisabled={!hasClusterNamespaceSelected}
                                        />
                                    </ToolbarItem>
                                    <ToolbarItem>
                                        <TimeWindowSelector
                                            activeTimeWindow={timeWindow}
                                            setActiveTimeWindow={setTimeWindow}
                                            isDisabled={isLoading || !hasClusterNamespaceSelected}
                                        />
                                    </ToolbarItem>
                                </ToolbarGroup>
                                <Divider orientation={{ default: 'vertical' }} />
                                <ToolbarGroup className="pf-u-flex-grow-1">
                                    <ToolbarItem className="pf-u-flex-grow-1">
                                        <NetworkSearch
                                            selectedCluster={clusterFromUrl.name}
                                            selectedNamespaces={namespacesFromUrl}
                                            selectedDeployments={deploymentsFromUrl}
                                            isDisabled={!hasClusterNamespaceSelected}
                                        />
                                    </ToolbarItem>
                                    <ToolbarItem>
                                        <DisplayOptionsSelect
                                            selectedOptions={displayOptions}
                                            setSelectedOptions={setDisplayOptions}
                                            isDisabled={!hasClusterNamespaceSelected}
                                        />
                                    </ToolbarItem>
                                </ToolbarGroup>
                                <ToolbarGroup alignment={{ default: 'alignRight' }}>
                                    <Divider
                                        component="div"
                                        orientation={{ default: 'vertical' }}
                                    />
                                    <ToolbarItem className="pf-u-color-200">
                                        <NodeUpdateSection
                                            isLoading={isLoading}
                                            lastUpdatedTime={lastUpdatedTime}
                                            nodeUpdatesCount={nodeUpdatesCount}
                                            updateNetworkNodes={updateNetworkNodes}
                                        />
                                    </ToolbarItem>
                                </ToolbarGroup>
                            </ToolbarContent>
                        </Toolbar>
                    </PageSection>
                    <Divider component="div" />
                </>
            )}
            <PageSection
                className="network-graph"
                variant={hasClusterNamespaceSelected ? 'default' : 'light'}
                padding={{ default: 'noPadding' }}
            >
                {hasReadAccessForGenerator && !hasClusterNamespaceSelected && (
                    <Drawer isExpanded={simulation.isOn} isInline>
                        <DrawerContent
                            panelContent={
                                <DrawerPanelContent
                                    className="cluster-simulation-drawer-panel"
                                    isResizable
                                    defaultSize="500px"
                                >
                                    <Button
                                        className="pf-topology-side-bar__dismiss"
                                        variant="plain"
                                        onClick={closeSimulatorSidebar}
                                        aria-label="Close sidebar"
                                    >
                                        <TimesIcon />
                                    </Button>
                                    <NetworkPolicySimulatorSidePanel
                                        simulator={simulator}
                                        setNetworkPolicyModification={setNetworkPolicyModification}
                                        scopeHierarchy={scopeHierarchy}
                                        scopeDeploymentCount={deploymentCount ?? 0}
                                    />
                                </DrawerPanelContent>
                            }
                        >
                            <DrawerContentBody>
                                <EmptyUnscopedState />
                            </DrawerContentBody>
                        </DrawerContent>
                    </Drawer>
                )}
                {models.activeModel.nodes.length > 0 &&
                    models.extraneousModel.nodes.length > 0 &&
                    !isLoading &&
                    hasClusterNamespaceSelected && (
                        <NetworkGraphContainer
                            models={models}
                            edgeState={edgeState}
                            displayOptions={displayOptions}
                            simulation={simulation}
                            clusterDeploymentCount={deploymentCount || 0}
                            scopeHierarchy={scopeHierarchy}
                        />
                    )}
                {isLoading && (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                )}
                <CIDRFormModal
                    selectedClusterId={selectedClusterId || ''}
                    isOpen={isCIDRBlockFormOpen}
                    onClose={toggleCIDRBlockForm}
                />
            </PageSection>
        </>
    );
}

export default NetworkGraphPage;
