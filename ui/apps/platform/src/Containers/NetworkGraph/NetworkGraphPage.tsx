import React, { useState, useEffect, useCallback } from 'react';
import {
    PageSection,
    Title,
    Bullseye,
    Spinner,
    Button,
    Divider,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import useDeepCompareEffect from 'use-deep-compare-effect';

import { timeWindows } from 'constants/timeWindows';
import useFetchClusters from 'hooks/useFetchClusters';
import useFetchDeploymentCount from 'hooks/useFetchDeploymentCount';
import useURLSearch from 'hooks/useURLSearch';
import { fetchNetworkFlowGraph, fetchNetworkPolicyGraph } from 'services/NetworkService';
import queryService from 'utils/queryService';
import timeWindowToDate from 'utils/timeWindows';
import { isCompleteSearchFilter } from 'utils/searchUtils';

import PageTitle from 'Components/PageTitle';
import useURLParameter from 'hooks/useURLParameter';
import EmptyUnscopedState from './components/EmptyUnscopedState';
import NetworkBreadcrumbs from './components/NetworkBreadcrumbs';
import NetworkSearch from './components/NetworkSearch';
import SimulateNetworkPolicyButton from './simulation/SimulateNetworkPolicyButton';
import EdgeStateSelect, { EdgeState } from './components/EdgeStateSelect';
import DisplayOptionsSelect, { DisplayOption } from './components/DisplayOptionsSelect';
import TimeWindowSelector from './components/TimeWindowSelector';
import NetworkGraph from './NetworkGraph';
import {
    transformPolicyData,
    transformActiveData,
    createExtraneousFlowsModel,
    graphModel,
} from './utils/modelUtils';
import getScopeHierarchy from './utils/getScopeHierarchy';
import getSimulation from './utils/getSimulation';
import {
    CustomEdgeModel,
    CustomModel,
    CustomNodeModel,
    DeploymentData,
} from './types/topology.type';

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

function NetworkGraphPage() {
    const [edgeState, setEdgeState] = useState<EdgeState>('active');
    const [displayOptions, setDisplayOptions] = useState<DisplayOption[]>([
        'policyStatusBadge',
        'externalBadge',
        'edgeLabel',
    ]);
    const [activeModel, setActiveModel] = useState<CustomModel>(emptyModel);
    const [extraneousFlowsModel, setExtraneousFlowsModel] = useState<CustomModel>(emptyModel);
    const [model, setModel] = useState<CustomModel>(emptyModel);
    const [isLoading, setIsLoading] = useState(false);
    const [timeWindow, setTimeWindow] = useState<typeof timeWindows[number]>(timeWindows[0]);
    const [lastUpdatedTime, setLastUpdatedTime] = useState<string>('never');

    const { searchFilter } = useURLSearch();
    const [simulationQueryValue] = useURLParameter('simulation', undefined);

    const {
        cluster: clusterFromUrl,
        namespaces: namespacesFromUrl,
        deployments: deploymentsFromUrl,
        remainingQuery,
    } = getScopeHierarchy(searchFilter);
    const simulation = getSimulation(simulationQueryValue);

    const hasClusterNamespaceSelected = Boolean(clusterFromUrl && namespacesFromUrl.length);

    const { clusters } = useFetchClusters();
    const selectedClusterId = clusters.find((cl) => cl.name === clusterFromUrl)?.id;
    const selectedCluster = { name: clusterFromUrl, id: selectedClusterId };
    const { deploymentCount } = useFetchDeploymentCount(selectedClusterId || '');

    useDeepCompareEffect(() => {
        // check that user is finished adding a complete filter
        const isQueryFilterComplete = isCompleteSearchFilter(remainingQuery);

        // only refresh the graph data from the API if both a cluster and at least one namespace are selected
        const isClusterNamespaceSelected =
            clusterFromUrl && namespacesFromUrl.length > 0 && deploymentCount;

        if (isQueryFilterComplete && isClusterNamespaceSelected) {
            if (selectedClusterId) {
                setIsLoading(true);

                const queryToUse = queryService.objectToWhereClause(remainingQuery);
                const timestampToUse = timeWindowToDate(timeWindow);

                Promise.all([
                    fetchNetworkFlowGraph(
                        selectedClusterId,
                        namespacesFromUrl,
                        deploymentsFromUrl,
                        queryToUse,
                        timestampToUse || undefined,
                        includePorts,
                        ALWAYS_SHOW_ORCHESTRATOR_COMPONENTS
                    ),
                    fetchNetworkPolicyGraph(
                        selectedClusterId,
                        namespacesFromUrl,
                        deploymentsFromUrl,
                        queryToUse,
                        undefined,
                        includePorts,
                        ALWAYS_SHOW_ORCHESTRATOR_COMPONENTS
                    ),
                ])
                    .then((values) => {
                        // get policy nodes from api response
                        const { nodes: policyNodes } = values[1].response;
                        // transform policy data to DataModel
                        const { policyDataModel, policyNodeMap } = transformPolicyData(
                            policyNodes,
                            deploymentCount || 0
                        );
                        // get active nodes from api response
                        const { nodes: activeNodes } = values[0].response;
                        // transform active data to DataModel
                        const { activeDataModel, activeEdgeMap, activeNodeMap } =
                            transformActiveData(activeNodes, policyNodeMap);

                        // create extraneous flows graph
                        const extraneousFlowsDataModel = createExtraneousFlowsModel(
                            policyDataModel,
                            activeNodeMap,
                            activeEdgeMap
                        );
                        setActiveModel(activeDataModel);
                        setExtraneousFlowsModel(extraneousFlowsDataModel);

                        const newUpdatedTimestamp = new Date();
                        // show only hours and minutes, use options with the default locale - use an empty array
                        const lastUpdatedDisplayTime = newUpdatedTimestamp.toLocaleTimeString([], {
                            hour: '2-digit',
                            minute: '2-digit',
                        });
                        setLastUpdatedTime(lastUpdatedDisplayTime);
                    })
                    .catch(() => {
                        // TODO
                    })
                    .finally(() => setIsLoading(false));
            }
        }
    }, [
        clusterFromUrl,
        namespacesFromUrl,
        deploymentsFromUrl,
        deploymentCount,
        remainingQuery,
        timeWindow,
    ]);

    const setModelByEdgeState = useCallback(() => {
        if (edgeState === 'active') {
            setModel(activeModel);
        } else if (edgeState === 'extraneous') {
            setModel(extraneousFlowsModel);
        }
    }, [edgeState, activeModel, extraneousFlowsModel]);

    useEffect(() => {
        setModelByEdgeState();
    }, [setModelByEdgeState]);

    useEffect(() => {
        const showPolicyState = !!displayOptions.includes('policyStatusBadge');
        const showExternalState = !!displayOptions.includes('externalBadge');
        const showEdgeLabels = !!displayOptions.includes('edgeLabel');
        let updatedNodes: CustomNodeModel[] = model.nodes;
        let updatedEdges: CustomEdgeModel[] = model.edges;

        // if all display options are true, set back to existing default data model
        if (showPolicyState && showExternalState && showEdgeLabels) {
            setModelByEdgeState();
        } else {
            // this is to update the display options visually for deployment nodes on the graph
            if (model.nodes?.length) {
                // need to improve perf to only perform this if policyStatusBadge OR externalBadge has changed
                updatedNodes = model.nodes.map((node) => {
                    const { data } = node;
                    if (data.type === 'DEPLOYMENT') {
                        return {
                            ...node,
                            data: {
                                ...data,
                                showPolicyState,
                                showExternalState,
                            } as DeploymentData,
                        };
                    }
                    return node;
                });
            }

            if (model.edges?.length) {
                // need to improve perf to only perform this if edgeLabel has changed
                updatedEdges = model.edges.map((edge) => {
                    const { data } = edge;
                    const { properties } = data;
                    return {
                        ...edge,
                        data: {
                            ...data,
                            properties,
                            tag: showEdgeLabels ? data.portProtocolLabel : undefined,
                        },
                    };
                });
            }

            const updatedModel: CustomModel = {
                ...model,
                nodes: updatedNodes,
                edges: updatedEdges,
            };
            setModel(updatedModel);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [displayOptions]);

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
                                selectedCluster={selectedCluster}
                                selectedNamespaces={namespacesFromUrl}
                                selectedDeployments={deploymentsFromUrl}
                            />
                        </ToolbarGroup>
                        <ToolbarGroup variant="button-group" alignment={{ default: 'alignRight' }}>
                            <ToolbarItem spacer={{ default: 'spacerMd' }}>
                                <Button variant="secondary">Manage CIDR blocks</Button>
                            </ToolbarItem>
                            <ToolbarItem spacer={{ default: 'spacerNone' }}>
                                <SimulateNetworkPolicyButton simulation={simulation} />
                            </ToolbarItem>
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Toolbar data-testid="network-graph-toolbar">
                    <ToolbarContent>
                        <ToolbarGroup variant="filter-group">
                            <ToolbarItem spacer={{ default: 'spacerMd' }}>
                                <EdgeStateSelect
                                    edgeState={edgeState}
                                    setEdgeState={setEdgeState}
                                />
                            </ToolbarItem>
                            <ToolbarItem>
                                <TimeWindowSelector
                                    activeTimeWindow={timeWindow}
                                    setActiveTimeWindow={setTimeWindow}
                                    isDisabled={isLoading}
                                />
                            </ToolbarItem>
                        </ToolbarGroup>
                        <ToolbarGroup className="pf-u-flex-grow-1">
                            <ToolbarItem className="pf-u-flex-grow-1">
                                <NetworkSearch
                                    selectedCluster={clusterFromUrl}
                                    selectedNamespaces={namespacesFromUrl}
                                    selectedDeployments={deploymentsFromUrl}
                                />
                            </ToolbarItem>
                            <ToolbarItem>
                                <DisplayOptionsSelect
                                    selectedOptions={displayOptions}
                                    setSelectedOptions={setDisplayOptions}
                                />
                            </ToolbarItem>
                        </ToolbarGroup>
                        <ToolbarGroup alignment={{ default: 'alignRight' }}>
                            <Divider component="div" orientation={{ default: 'vertical' }} />
                            <ToolbarItem>Last updated at {lastUpdatedTime}</ToolbarItem>
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="network-graph"
                variant={hasClusterNamespaceSelected ? 'default' : 'light'}
                padding={{ default: 'noPadding' }}
            >
                {!hasClusterNamespaceSelected && <EmptyUnscopedState />}
                {model.nodes.length > 0 && !isLoading && (
                    <NetworkGraph
                        model={model}
                        edgeState={edgeState}
                        simulation={simulation}
                        selectedClusterId={selectedClusterId || ''}
                    />
                )}
                {isLoading && (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                )}
            </PageSection>
        </>
    );
}

export default NetworkGraphPage;
