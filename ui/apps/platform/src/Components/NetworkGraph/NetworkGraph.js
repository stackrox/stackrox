import React, { useState, useRef, useEffect } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { withRouter } from 'react-router-dom';
import Cytoscape from 'cytoscape';
import CytoscapeComponent from 'react-cytoscapejs';
import { throttle, debounce, includes } from 'lodash';
import popper from 'cytoscape-popper';
/* Cannot use neither Tooltip nor HoverHint components as Cytoscape renders on
canvas (no DOM elements). Instead using 'cytoscape-popper' and  special
configuration of 'tippy.js' instance to position the tooltip. */
// eslint-disable-next-line no-restricted-imports
import tippy from 'tippy.js';
import ReactDOM from 'react-dom';

import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import { actions as graphActions } from 'reducers/network/graph';
import GraphLoader from 'Containers/Network/Graph/Overlays/GraphLoader';
import { edgeGridLayout, getParentPositions } from 'Containers/Network/Graph/networkGraphLayouts';
import { NS_FONT_SIZE, MAX_ZOOM, MIN_ZOOM, ZOOM_STEP, GRAPH_PADDING } from 'constants/networkGraph';
import { filterModes } from 'constants/networkFilterModes';
import entityTypes from 'constants/entityTypes';
import style from 'Containers/Network/Graph/networkGraphStyles';
import {
    getLinks,
    isNamespace,
    isNamespaceEdge,
    getNodeData,
    getEdges,
    getNamespaceEdgeNodes,
    getNamespaceList,
    getDeploymentList,
    getFilteredNodes,
    getNetworkFlows,
    getEdgesFromNode,
    getIngressPortsAndProtocols,
    getEgressPortsAndProtocols,
    edgeTypes,
} from 'utils/networkGraphUtils';
import { knownBackendFlags, isBackendFeatureFlagEnabled } from 'utils/featureFlags';

import { defaultTippyTooltipProps } from '@stackrox/ui-components/lib/Tooltip';
import NodeTooltipOverlay from './NodeTooltipOverlay';
import NamespaceEdgeTooltipOverlay from './NamespaceEdgeTooltipOverlay';
import EdgeTooltipOverlay from './EdgeTooltipOverlay';

Cytoscape.use(popper);
Cytoscape('layout', 'edgeGridLayout', edgeGridLayout);
Cytoscape.use(edgeGridLayout);

const NetworkGraph = ({
    networkEdgeMap,
    networkNodeMap,
    onNodeClick,
    onNamespaceClick,
    onClickOutside,
    filterState,
    setNetworkGraphRef,
    setSelectedNamespace,
    setSelectedNodeInGraph,
    simulatorOn,
    selectedClusterName,
    history,
    match,
    featureFlags,
}) => {
    const [selectedNode, setSelectedNode] = useState();
    const [hoveredElement, setHoveredElement] = useState();
    const [firstRenderFinished, setFirstRenderFinished] = useState(false);
    const nodeSideMapRef = useRef({});
    const zoomFontMapRef = useRef({});
    const nodeSideMap = nodeSideMapRef.current;
    const zoomFontMap = zoomFontMapRef.current;
    const cyRef = useRef();
    const tippyRef = useRef();
    const namespacesWithDeployments = {};

    const nodes = getFilteredNodes(networkNodeMap, filterState);
    const data = nodes.map((datum) => ({
        ...datum,
        isActive: filterState !== filterModes.active && datum.internetAccess,
    }));

    const links = getLinks(data, networkEdgeMap, networkNodeMap, filterState);

    // @TODO: Remove "showPortsAndProtocols" when the feature flag "ROX_NETWORK_GRAPH_PORTS" is defaulted to true
    const showPortsAndProtocols = isBackendFeatureFlagEnabled(
        featureFlags,
        knownBackendFlags.ROX_NETWORK_GRAPH_PORTS,
        false
    );

    function showTooltip(elm, component) {
        if (!elm || !component || !cyRef) {
            return;
        }
        hideTooltip();

        try {
            const popperRef = elm.popperRef();
            const content = document.createElement('div');
            ReactDOM.render(component, content);

            tippyRef.current = tippy(document.createElement('div'), {
                content,
                ...defaultTippyTooltipProps,
                getReferenceClientRect: popperRef.getBoundingClientRect,
                onHidden(instance) {
                    instance.destroy();
                },
            });

            tippyRef.current.show();
        } catch (err) {
            mouseOutHandler();
        }
    }

    function hideTooltip() {
        if (tippyRef.current) {
            tippyRef.current.destroy();
        }
    }

    function nodeHoverHandler(ev) {
        const node = ev.target.data();
        const { id, name, parent, listenPorts, side, type } = node;
        const isDeployment = type === entityTypes.DEPLOYMENT;
        if (!cyRef || !isDeployment || side) {
            return;
        }

        setHoveredElement(node);

        const configObj = { ...getConfigObj(), hoveredNode: node };
        const edgesFromNode = getEdgesFromNode(configObj);
        const { networkFlows, numIngressFlows, numEgressFlows } = getNetworkFlows(
            edgesFromNode,
            filterState
        );
        const ingressPortsAndProtocols = getIngressPortsAndProtocols(networkFlows);
        const egressPortsAndProtocols = getEgressPortsAndProtocols(networkFlows);

        const nodeElm = cyRef.current.getElementById(id);
        const parentElm = cyRef.current.getElementById(parent);

        const component = (
            <NodeTooltipOverlay
                deploymentName={name}
                numIngressFlows={numIngressFlows}
                numEgressFlows={numEgressFlows}
                ingressPortsAndProtocols={ingressPortsAndProtocols}
                egressPortsAndProtocols={egressPortsAndProtocols}
                listenPorts={listenPorts}
                showPortsAndProtocols={showPortsAndProtocols}
            />
        );

        showTooltip(nodeElm, component);
        const children = parentElm.descendants();
        children.removeClass('background');
    }

    function edgeHoverHandler(ev) {
        const edge = ev.target.data();
        const { id, portsAndProtocols } = edge;
        const edgeElm = cyRef.current.getElementById(id);
        let component;
        if (
            !cyRef ||
            (edge.source === hoveredElement?.source && edge.target === hoveredElement?.target)
        ) {
            return;
        }

        setHoveredElement(edge);

        if (isNamespaceEdge(edge)) {
            const {
                numBidirectionalLinks,
                numUnidirectionalLinks,
                numActiveBidirectionalLinks,
                numActiveUnidirectionalLinks,
                numAllowedBidirectionalLinks,
                numAllowedUnidirectionalLinks,
            } = edge;
            component = (
                <NamespaceEdgeTooltipOverlay
                    numBidirectionalLinks={numBidirectionalLinks}
                    numUnidirectionalLinks={numUnidirectionalLinks}
                    numActiveBidirectionalLinks={numActiveBidirectionalLinks}
                    numActiveUnidirectionalLinks={numActiveUnidirectionalLinks}
                    numAllowedBidirectionalLinks={numAllowedBidirectionalLinks}
                    numAllowedUnidirectionalLinks={numAllowedUnidirectionalLinks}
                    portsAndProtocols={portsAndProtocols}
                    filterState={filterState}
                    showPortsAndProtocols={showPortsAndProtocols}
                />
            );
        } else {
            const { sourceNodeName, targetNodeName, isBidirectional } = edge;
            component = (
                <EdgeTooltipOverlay
                    source={sourceNodeName}
                    target={targetNodeName}
                    isBidirectional={isBidirectional}
                    portsAndProtocols={portsAndProtocols}
                    showPortsAndProtocols={showPortsAndProtocols}
                />
            );
        }
        showTooltip(edgeElm, component);
    }

    function mouseOutHandler() {
        hideTooltip();
        setHoveredElement();
    }

    function nodeMouseOutHandler() {
        if (hoveredElement?.type === 'DEPLOYMENT') {
            mouseOutHandler();
        }
    }

    function edgeMouseOutHandler() {
        if (includes(Object.values(edgeTypes), hoveredElement?.type)) {
            mouseOutHandler();
        }
    }

    function clickHandler(ev) {
        const { target } = ev;
        const evData = target.data && target.data();
        const { id, type } = evData;
        const targetIsNamespace = type === entityTypes.NAMESPACE;
        const isEdge = target.isEdge && target.isEdge();

        // Canvas or Selected node click: clear selection
        if (
            !id ||
            !evData ||
            (selectedNode && id === selectedNode.id) ||
            type === entityTypes.CLUSTER
        ) {
            setSelectedNode();
            setSelectedNodeInGraph();
            onClickOutside();
            history.push('/main/network');
            return;
        }

        // Edge click or edge node click
        if (isEdge || evData.side) {
            return;
        }

        // Namespace Click
        if (targetIsNamespace) {
            if (id) {
                const deployments = (namespacesWithDeployments[id] || []).map((deployment) => {
                    const deploymentEdges = getEdgesFromNode({
                        ...getConfigObj(),
                        selectedNode: deployment.data,
                    });
                    const modifiedDeployment = {
                        ...deployment,
                    };
                    modifiedDeployment.data.edges = deploymentEdges;
                    return modifiedDeployment;
                });
                onNamespaceClick({ id, deployments });
                setSelectedNode(evData);
                setSelectedNodeInGraph(evData);
            }
            return;
        }

        // if we didn't return early, must be click off a NS
        setSelectedNamespace(null);

        // New Node click: select node
        if (target.isNode()) {
            setSelectedNode(evData);
            setSelectedNodeInGraph(evData);
            history.push(`/main/network/${id}`);
            onNodeClick(evData);
        }
    }

    function zoomHandler() {
        if (!cyRef || !cyRef.current) {
            return;
        }

        // to dynamically set the font size of namespace labels
        const zoomConstant = 20;
        const curZoomLevel = Math.round(cyRef.current.zoom() * zoomConstant);
        if (!zoomFontMap[curZoomLevel]) {
            zoomFontMap[curZoomLevel] = Math.max(
                (NS_FONT_SIZE / curZoomLevel) * zoomConstant,
                NS_FONT_SIZE
            );
        }
        cyRef.current.nodes(':parent').style('font-size', zoomFontMap[curZoomLevel]);
        cyRef.current.edges('.namespace').style('font-size', zoomFontMap[curZoomLevel]);
    }

    function zoomToFit() {
        if (!cyRef) {
            return;
        }
        cyRef.current.fit(null, GRAPH_PADDING);
        const newMinZoom = Math.min(cyRef.current.zoom(), cyRef.current.minZoom());
        cyRef.current.minZoom(newMinZoom);
        zoomHandler();
    }

    function zoomIn() {
        if (!cyRef.current) {
            return;
        }

        cyRef.current.zoom({
            level: Math.max(cyRef.current.zoom() + ZOOM_STEP, cyRef.current.minZoom()),
            renderedPosition: { x: cyRef.current.width() / 2, y: cyRef.current.height() / 2 },
        });
    }

    function zoomOut() {
        if (!cyRef.current) {
            return;
        }

        cyRef.current.zoom({
            level: Math.min(cyRef.current.zoom() - ZOOM_STEP, MAX_ZOOM),
            renderedPosition: { x: cyRef.current.width() / 2, y: cyRef.current.height() / 2 },
        });
    }

    function getConfigObj() {
        const hoveredNode = hoveredElement?.type === 'DEPLOYMENT' ? hoveredElement : null;
        const hoveredEdge = includes(Object.values(edgeTypes), hoveredElement?.type)
            ? hoveredElement
            : null;
        return {
            hoveredNode,
            selectedNode,
            hoveredEdge,
            links,
            nodes,
            filterState,
            nodeSideMap,
            networkNodeMap,
            featureFlags,
        };
    }

    function getNodeDataFromList(id) {
        const configObj = getConfigObj();
        // for the case when you want to pull the selected node from the URL on refresh
        if (match.params.deploymentId) {
            configObj.selectedNode = { id: match.params.deploymentId };
        }
        const filteredData = data.filter((datum) => datum?.entity?.deployment);
        const deploymentList = getDeploymentList(filteredData, configObj);
        return getNodeData(id, deploymentList);
    }

    function getElements() {
        const configObj = getConfigObj();
        const filteredData = data.filter((datum) => datum?.entity?.deployment);
        const deploymentList = getDeploymentList(filteredData, configObj);
        const namespaceList = getNamespaceList(
            filteredData,
            deploymentList,
            configObj,
            selectedClusterName
        );
        const namespaceEdgeNodes = getNamespaceEdgeNodes(namespaceList);

        namespaceList.forEach((namespace) => {
            deploymentList.forEach((deployment) => {
                if (!namespacesWithDeployments[namespace.data.id]) {
                    namespacesWithDeployments[namespace.data.id] = [];
                }
                if (deployment.data.parent === namespace.data.id) {
                    namespacesWithDeployments[namespace.data.id].push(deployment);
                }
            });
        });

        const clusterGroup = {
            classes: 'cluster',
            data: {
                id: selectedClusterName,
                name: selectedClusterName,
                active: false,
                type: entityTypes.CLUSTER,
            },
        };

        const allNodes = [...namespaceList, ...deploymentList, ...namespaceEdgeNodes];
        const allEdges = getEdges(configObj);

        // @TODO: Remove "showClusters" flag when the feature flag "ROX_NETWORK_GRAPH_EXTERNAL_SRCS" is defaulted to true
        //        and don't forget to add `clusterGroup to `allNodes` array above
        const showCluster = isBackendFeatureFlagEnabled(
            featureFlags,
            knownBackendFlags.ROX_NETWORK_GRAPH_EXTERNAL_SRCS,
            false
        );
        if (showCluster) {
            allNodes.push(clusterGroup);
        }
        // end of TODO to be removed

        return {
            nodes: allNodes,
            edges: allEdges,
        };
    }

    // Calculate which namespace box side combinations are shortest and store them
    function calculateNodeSideMap(changedNodeId) {
        if (!cyRef.current) {
            return;
        }

        // Get a map of all the side nodes per namespace
        const groups = cyRef.current.nodes(':parent');
        const namespaces = groups.filter((group) => {
            const groupData = group.data();
            return groupData.type !== entityTypes.CLUSTER;
        });
        const sideNodesPerParent = namespaces.reduce((acc, namespace) => {
            const { id } = namespace.data(); // to
            if (!id) {
                return { ...acc };
            }
            const sideNodes = cyRef.current.nodes(`[parent="${id}"][side]`);

            const nodesInfo = sideNodes.map((node) => {
                const { x, y } = node.position();
                const { side } = node.data();
                return {
                    node,
                    side,
                    id: node.id(),
                    x,
                    y,
                };
            });
            return { ...acc, [id]: nodesInfo };
        }, {});

        const distances = {};

        function getDistance(sourceSideNode, targetSideNode) {
            const key = [sourceSideNode.id, targetSideNode.id].sort().join('**__**');
            const cachedDistance = distances[key];
            if (cachedDistance) {
                return cachedDistance;
            }
            const dX = Math.abs(sourceSideNode.x - targetSideNode.x);
            const dY = Math.abs(sourceSideNode.y - targetSideNode.y);
            const distance = Math.sqrt(dX * dX + dY * dY);
            distances[key] = distance;
            return distance;
        }
        // for each namespace, go through each other namespace
        namespaces.forEach((sourceNS, i) => {
            const sourceName = sourceNS.data().id;
            const sourceSideNodes = sideNodesPerParent[sourceName];
            nodeSideMap[sourceName] = nodeSideMap[sourceName] || {};
            const sourceMap = nodeSideMap[sourceName];

            namespaces.forEach((targetNS, j) => {
                const targetName = targetNS.data().id;

                if (
                    i === j ||
                    (changedNodeId && ![sourceName, targetName].includes(changedNodeId))
                ) {
                    return;
                }

                const targetSideNodes = sideNodesPerParent[targetName];
                let shortest;
                // check distances between every combination of side nodes to find shortest
                sourceSideNodes.forEach((sourceSideNode) => {
                    const sourceSide = sourceSideNode.side;
                    const targetSideNode = targetSideNodes.find((tgtNode) => {
                        const { side } = tgtNode;
                        if (sourceSide === 'top') {
                            return side === 'bottom';
                        }
                        if (sourceSide === 'bottom') {
                            return side === 'top';
                        }
                        if (sourceSide === 'left') {
                            return side === 'right';
                        }
                        if (sourceSide === 'right') {
                            return side === 'left';
                        }
                        return false;
                    });

                    const distance = getDistance(sourceSideNode, targetSideNode);
                    if (!shortest || shortest.distance > distance) {
                        shortest = {
                            source: sourceSideNode.id,
                            target: targetSideNode.id,
                            sourceSide: sourceSideNode.side,
                            targetSide: targetSideNode.side,
                            distance,
                        };
                    }
                });
                sourceMap[targetName] = shortest;
            });
        });
    }

    function handleDrag(ev) {
        let changedNodeId;
        if (ev && ev.target) {
            changedNodeId = ev.target.data().id;
        }

        calculateNodeSideMap(changedNodeId);
        const configObj = getConfigObj();
        const newEdges = getEdges(configObj);

        cyRef.current.remove('edge');
        cyRef.current.add(newEdges);
    }

    function configureCY(cyInstance) {
        cyRef.current = cyInstance;

        cyRef.current
            .off('click mouseover mouseout mousedown drag')
            .on('click', clickHandler)
            .on('mouseover', 'node', throttle(nodeHoverHandler, 100))
            .on('mouseout mousedown', 'node', debounce(nodeMouseOutHandler, 100))
            .on('mouseover', 'edge', debounce(edgeHoverHandler, 200))
            .on('mouseout mousedown', 'edge', debounce(edgeMouseOutHandler, 100))
            .on('drag', throttle(handleDrag, 100))
            .on('zoom', zoomHandler)
            .ready(() => {
                if (firstRenderFinished) {
                    return;
                }
                zoomToFit();
                setFirstRenderFinished(true);
            });

        // if running in the UI e2e test environment, expose the cytoscape object to the tests
        if (window.Cypress) {
            window.cytoscape = cyRef.current;
        }
    }

    const elements = getElements();
    // Effects
    function setWindowResize() {
        window.addEventListener(
            'resize',
            throttle(() => zoomToFit, 100)
        );

        const cleanup = () => {
            window.removeEventListener('resize');
        };

        return cleanup;
    }

    function setGraphRef() {
        setNetworkGraphRef({
            zoomToFit,
            zoomIn,
            zoomOut,
            setSelectedNode,
            selectedNode,
            getNodeData: getNodeDataFromList,
            onNodeClick,
            getConfigObj,
        });
    }

    function runLayout() {
        if (!cyRef.current) {
            return;
        }
        const CY = cyRef.current;
        const NSPositions = getParentPositions(CY.nodes(), { x: 100, y: 100 }); // all nodes, padding

        NSPositions.forEach((position) => {
            const { id, x, y } = position;
            CY.layout({
                name: 'edgeGridLayout',
                parentPadding: { bottom: 5, top: 0, left: 0, right: 0 },
                position: { x, y },
                eles: CY.nodes(`[parent="${id}"]`),
            }).run();
        });
        CY.fit(null, GRAPH_PADDING);
        const node = getNodeDataFromList(match.params.deploymentId);
        if (setSelectedNodeInGraph && node.length) {
            setSelectedNodeInGraph(node[0].data);
            setSelectedNode(node[0].data);
            onNodeClick(node[0].data);
        }
        if (selectedNode && isNamespace(selectedNode)) {
            onNamespaceClick({
                id: selectedNode.id,
                deployments: namespacesWithDeployments[selectedNode.id] || [],
            });
        }
        if (simulatorOn) {
            setSelectedNode();
            setSelectedNodeInGraph();
            setSelectedNamespace(null);
        }
    }

    function grabifyNamespaces() {
        if (!cyRef.current) {
            return;
        }
        const CY = cyRef.current;
        CY.nodes(`.cluster`).ungrabify();
    }

    useEffect(setWindowResize, []);
    useEffect(setGraphRef, []);
    useEffect(runLayout, [networkNodeMap, networkEdgeMap, filterState, simulatorOn]);
    useEffect(grabifyNamespaces);
    useEffect(calculateNodeSideMap);

    const normalizedElements = CytoscapeComponent.normalizeElements(elements);

    const loader = !normalizedElements && (
        <div className="absolute flex h-full items-center justify-center top-0 w-full pointer-events-none">
            <GraphLoader isLoading />
        </div>
    );

    return (
        <div className="h-full w-full relative">
            <div
                id="cytoscapeContainer"
                className="w-full h-full cursor-pointer cytoscape-container"
            >
                <CytoscapeComponent
                    elements={normalizedElements}
                    layout={{ name: 'grid' }}
                    stylesheet={style}
                    cy={configureCY}
                    minZoom={MIN_ZOOM}
                    maxZoom={MAX_ZOOM}
                    style={{ width: '100%', height: '100%' }}
                />
            </div>
            {loader}
        </div>
    );
};

NetworkGraph.propTypes = {
    networkEdgeMap: PropTypes.shape({}),
    networkNodeMap: PropTypes.shape({}).isRequired,
    onNamespaceClick: PropTypes.func.isRequired,
    onNodeClick: PropTypes.func.isRequired,
    onClickOutside: PropTypes.func.isRequired,
    filterState: PropTypes.number.isRequired,
    setNetworkGraphRef: PropTypes.func.isRequired,
    setSelectedNamespace: PropTypes.func.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
    setSelectedNodeInGraph: PropTypes.func,
    simulatorOn: PropTypes.bool.isRequired,
    selectedClusterName: PropTypes.string.isRequired,
    featureFlags: PropTypes.arrayOf(PropTypes.shape),
};

NetworkGraph.defaultProps = {
    setSelectedNodeInGraph: null,
    networkEdgeMap: {},
    featureFlags: [],
};

const mapStateToProps = createStructuredSelector({
    featureFlags: selectors.getFeatureFlags,
});
const mapDispatchToProps = {
    setNetworkGraphRef: graphActions.setNetworkGraphRef,
    setSelectedNamespace: graphActions.setSelectedNamespace,
    setSelectedNodeInGraph: graphActions.setSelectedNode,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(NetworkGraph));
