import React, { useState, useRef, useEffect, useMemo } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import Cytoscape from 'cytoscape';
import CytoscapeComponent from 'react-cytoscapejs';
import debounce from 'lodash/debounce';
import includes from 'lodash/includes';
import throttle from 'lodash/throttle';
import popper from 'cytoscape-popper';
/* Cannot use neither Tooltip nor HoverHint components as Cytoscape renders on
canvas (no DOM elements). Instead using 'cytoscape-popper' and  special
configuration of 'tippy.js' instance to position the tooltip. */
// eslint-disable-next-line no-restricted-imports
import tippy from 'tippy.js';
import ReactDOM from 'react-dom';
import GraphLoader from 'Containers/Network/Graph/Overlays/GraphLoader';
import { edgeGridLayout, getParentPositions } from 'Containers/Network/Graph/networkGraphLayouts';
import {
    NS_FONT_SIZE,
    MAX_ZOOM,
    MIN_ZOOM,
    ZOOM_STEP,
    GRAPH_PADDING,
    OUTER_PADDING,
    OUTER_SPACING_FACTOR,
    nodeTypes,
} from 'constants/networkGraph';
import { filterModes } from 'constants/networkFilterModes';
import entityTypes from 'constants/entityTypes';

import style from 'Containers/Network/Graph/networkGraphStyles';
import { getLinks, getFilteredLinks } from 'utils/networkLink.utils';
import {
    getExternalEntitiesNode,
    getCIDRBlockNodes,
    getFilteredNodes,
} from 'utils/networkNode.utils';
import { getClusterNode } from 'utils/networkUtils';
import {
    getIsNamespaceEdge,
    getNodeData,
    getEdges,
    getNamespaceEdgeNodes,
    getNamespaceList,
    getExternalEntitiesEdgeNodes,
    getCIDRBlockEdgeNodes,
    getDeploymentList,
    getEdgesFromNode,
    getIngressPortsAndProtocols,
    getEgressPortsAndProtocols,
    edgeTypes,
    getIsNodeHoverable,
} from 'utils/networkGraphUtils';
import { getNetworkFlows } from 'utils/networkUtils/getNetworkFlows';

import { defaultTippyTooltipProps } from '@stackrox/ui-components/lib/Tooltip';
import NodeTooltipOverlay from './NodeTooltipOverlay';
import NamespaceEdgeTooltipOverlay from './NamespaceEdgeTooltipOverlay';
import EdgeTooltipOverlay from './EdgeTooltipOverlay';
import {
    enhanceNodesWithSimulatedStatus,
    enhanceLinksWithSimulatedStatus,
} from './baselineSimulationUtils';

// This was getting annoying when I kept making changes to the code and it would break
try {
    Cytoscape.use(popper);
} catch (error) {
    // popper already exists
}
Cytoscape('layout', 'edgeGridLayout', edgeGridLayout);
Cytoscape.use(edgeGridLayout);

const NetworkGraph = ({
    networkEdgeMap,
    networkNodeMap,
    onNodeClick,
    onNamespaceClick,
    onExternalEntitiesClick,
    onClickOutside,
    filterState,
    setNetworkGraphRef,
    setSelectedNamespace,
    setSelectedNodeInGraph,
    selectedClusterName,
    showNamespaceFlows,
    history,
    location,
    match,
    featureFlags,
    lastUpdatedTimestamp,
    selectedClusterId,
    isReadOnly,
    simulatedBaselines,
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

    const nodes = useMemo(() => {
        const filteredNodes = getFilteredNodes(networkNodeMap, filterState);
        return selectedNode && simulatedBaselines.length
            ? enhanceNodesWithSimulatedStatus(selectedNode, filteredNodes, simulatedBaselines)
            : filteredNodes;
    }, [networkNodeMap, filterState, selectedNode, simulatedBaselines]);

    const data = useMemo(() => {
        return nodes.map((datum) => ({
            ...datum,
            isActive: filterState !== filterModes.active && datum.internetAccess,
        }));
    }, [nodes, filterState]);

    const links = useMemo(() => {
        const result = getLinks(data, networkEdgeMap, networkNodeMap, filterState, featureFlags);
        return selectedNode && simulatedBaselines.length
            ? enhanceLinksWithSimulatedStatus(selectedNode, result, simulatedBaselines)
            : result;
    }, [
        data,
        networkEdgeMap,
        networkNodeMap,
        filterState,
        featureFlags,
        selectedNode,
        simulatedBaselines,
    ]);
    const filteredLinks = useMemo(() => getFilteredLinks(links), [links]);

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
        const { id, name, listenPorts, side, type } = node;
        const isNodeHoverable = getIsNodeHoverable(type);
        if (!cyRef || !isNodeHoverable || side) {
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

        const component = (
            <NodeTooltipOverlay
                deploymentName={name}
                numIngressFlows={numIngressFlows}
                numEgressFlows={numEgressFlows}
                ingressPortsAndProtocols={ingressPortsAndProtocols}
                egressPortsAndProtocols={egressPortsAndProtocols}
                listenPorts={listenPorts}
            />
        );

        showTooltip(nodeElm, component);
    }

    function edgeHoverHandler(ev) {
        const edge = ev.target.data();
        const { id, portsAndProtocols, type } = edge;
        const edgeElm = cyRef.current.getElementById(id);
        let component;
        if (
            !cyRef ||
            (edge.source === hoveredElement?.source && edge.target === hoveredElement?.target)
        ) {
            return;
        }

        setHoveredElement(edge);

        if (getIsNamespaceEdge(type)) {
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
                />
            );
        }
        showTooltip(edgeElm, component);
    }

    function mouseOutHandler() {
        hideTooltip();
        setHoveredElement();
    }

    function clickHandler(ev) {
        if (!isReadOnly) {
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
                history.push(`/main/network${location.search}`);
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
            setSelectedNamespace();

            // New Node click: select node
            if (target.isNode()) {
                setSelectedNode(evData);
                setSelectedNodeInGraph(evData);

                if (type === nodeTypes.EXTERNAL_ENTITIES || type === nodeTypes.CIDR_BLOCK) {
                    onExternalEntitiesClick();
                } else {
                    history.push(`/main/network/${id}${location.search}`);
                    onNodeClick(evData);
                }
            }
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
        const hoveredNode = getIsNodeHoverable(hoveredElement?.type) ? hoveredElement : null;
        const hoveredEdge = includes(Object.values(edgeTypes), hoveredElement?.type)
            ? hoveredElement
            : null;

        const shouldShowNamespaceEdges =
            !!hoveredNode || !!hoveredEdge || !!selectedNode || showNamespaceFlows === 'show';

        return {
            hoveredNode,
            selectedNode,
            hoveredEdge,
            unfilteredLinks: links,
            links: filteredLinks,
            nodes,
            filterState,
            nodeSideMap,
            networkNodeMap,
            featureFlags,
            shouldShowNamespaceEdges,
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
            selectedClusterName,
            filterState
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

        let allNodes = [...namespaceList, ...deploymentList, ...namespaceEdgeNodes];
        const allEdges = getEdges(configObj);

        const clusterNode = getClusterNode(selectedClusterName);
        allNodes.push(clusterNode);

        const externalEntitiesNode = getExternalEntitiesNode(data, configObj);
        if (externalEntitiesNode) {
            const externalEntitiesEdgeNodes = getExternalEntitiesEdgeNodes(externalEntitiesNode);

            allNodes = allNodes.concat(externalEntitiesEdgeNodes, externalEntitiesNode);
        }

        const cidrBlockNodes = getCIDRBlockNodes(data, configObj);
        if (cidrBlockNodes?.length) {
            const cidrBlockEdgeNodes = getCIDRBlockEdgeNodes(cidrBlockNodes);

            allNodes = allNodes.concat(cidrBlockEdgeNodes, cidrBlockNodes);
        }

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
        const parents = groups.filter((group) => {
            return (
                group.data().type === entityTypes.NAMESPACE ||
                group.data().type === nodeTypes.EXTERNAL_ENTITIES ||
                group.data().type === nodeTypes.CIDR_BLOCK
            );
        });
        const sideNodesPerParent = parents.reduce((acc, parent) => {
            const { id } = parent.data(); // to
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
        // for each parent, go through each other parents
        parents.forEach((source, i) => {
            const sourceName = source.data().id;
            const sourceSideNodes = sideNodesPerParent[sourceName];
            nodeSideMap[sourceName] = nodeSideMap[sourceName] || {};
            const sourceMap = nodeSideMap[sourceName];

            parents.forEach((target, j) => {
                const targetName = target.data().id;

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
    }

    // This is called once we stop dragging
    function freeDrag() {
        // This should update the network graph after we finish dragging
        setHoveredElement(null);
    }

    function configureCY(cyInstance) {
        cyRef.current = cyInstance;

        cyRef.current
            .off('click mouseover mouseout mousedown drag')
            .on('click', clickHandler)
            .on('mouseover', 'node', debounce(nodeHoverHandler, 200))
            .on('mouseover', 'edge', debounce(edgeHoverHandler, 200))
            .on('mouseout mousedown', debounce(mouseOutHandler, 100))
            .on('drag', throttle(handleDrag, 100))
            .on('free', freeDrag)
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

        if (match.params.externalType) {
            const els = getElements();
            const externalNode = els?.nodes?.find((node) => {
                return node?.data?.id === match.params.deploymentId;
            });
            const { data: externalNodeData } = externalNode;
            setSelectedNode(externalNodeData);
            setSelectedNodeInGraph(externalNodeData);

            onExternalEntitiesClick();
        }
        const node = getNodeDataFromList(match.params.deploymentId);
        if (setSelectedNodeInGraph && node.length) {
            setSelectedNodeInGraph(node[0].data);
            setSelectedNode(node[0].data);
            onNodeClick(node[0].data);
        }
    }

    function grabifyNamespaces() {
        if (!cyRef.current) {
            return;
        }
        const CY = cyRef.current;
        CY.nodes(`.cluster`).ungrabify();
        CY.nodes(`.deployment`).ungrabify();
    }

    /* eslint-disable react-hooks/exhaustive-deps */
    useEffect(setWindowResize, []);
    useEffect(setGraphRef, []);
    useEffect(runLayout, [
        networkNodeMap,
        networkEdgeMap,
        filterState,
        isReadOnly,
        lastUpdatedTimestamp,
        match.params.deploymentId,
        simulatedBaselines,
    ]);
    /* eslint-enable react-hooks/exhaustive-deps */
    useEffect(grabifyNamespaces);
    useEffect(calculateNodeSideMap);

    const normalizedElements = CytoscapeComponent.normalizeElements(elements);

    return (
        <div className="h-full w-full relative network-grid-bg">
            <div
                id="cytoscapeContainer"
                className="w-full h-full cursor-pointer cytoscape-container"
            >
                <CytoscapeComponent
                    key={selectedClusterId}
                    elements={normalizedElements}
                    layout={{
                        name: 'grid',
                        padding: OUTER_PADDING,
                        spacingFactor: OUTER_SPACING_FACTOR,
                    }}
                    stylesheet={style}
                    cy={configureCY}
                    minZoom={MIN_ZOOM}
                    maxZoom={MAX_ZOOM}
                    style={{ width: '100%', height: '100%' }}
                />
            </div>
            {!normalizedElements && (
                <div className="absolute flex h-full items-center justify-center top-0 w-full pointer-events-none">
                    <GraphLoader />
                </div>
            )}
        </div>
    );
};

NetworkGraph.propTypes = {
    networkEdgeMap: PropTypes.shape({}),
    networkNodeMap: PropTypes.shape({}).isRequired,
    onNamespaceClick: PropTypes.func.isRequired,
    onExternalEntitiesClick: PropTypes.func.isRequired,
    onNodeClick: PropTypes.func.isRequired,
    onClickOutside: PropTypes.func.isRequired,
    filterState: PropTypes.number.isRequired,
    setNetworkGraphRef: PropTypes.func.isRequired,
    setSelectedNamespace: PropTypes.func.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
    setSelectedNodeInGraph: PropTypes.func,
    isReadOnly: PropTypes.bool,
    selectedClusterName: PropTypes.string.isRequired,
    showNamespaceFlows: PropTypes.string.isRequired,
    featureFlags: PropTypes.arrayOf(PropTypes.shape),
    lastUpdatedTimestamp: PropTypes.instanceOf(Date),
    selectedClusterId: PropTypes.string,
    simulatedBaselines: PropTypes.arrayOf(PropTypes.shape({})),
};

NetworkGraph.defaultProps = {
    setSelectedNodeInGraph: null,
    networkEdgeMap: {},
    featureFlags: [],
    lastUpdatedTimestamp: null,
    selectedClusterId: null,
    isReadOnly: false,
    simulatedBaselines: [],
};

export default withRouter(NetworkGraph);
