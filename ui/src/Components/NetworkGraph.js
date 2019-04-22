import React, { useState, useRef, useEffect } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { actions as graphActions } from 'reducers/network/graph';

import GraphLoader from 'Containers/Network/Graph/Overlays/GraphLoader';

import Cytoscape from 'cytoscape';
import CytoscapeComponent from 'react-cytoscapejs';
import popper from 'cytoscape-popper';
import Tippy from 'tippy.js';
import { uniq, throttle } from 'lodash';

import { edgeGridLayout, getParentPositions } from 'Containers/Network/Graph/networkGraphLayouts';
import { filterModes } from 'Containers/Network/Graph/filterModes';
import style from 'Containers/Network/Graph/networkGraphStyles';
import { getLinks, nonIsolated } from 'utils/networkGraphUtils';
import { NS_FONT_SIZE, MAX_ZOOM, MIN_ZOOM, ZOOM_STEP, GRAPH_PADDING } from 'constants/networkGraph';

function getClasses(map) {
    return Object.entries(map)
        .filter(entry => entry[1])
        .map(entry => entry[0])
        .join(' ');
}

Cytoscape.use(popper);
Cytoscape('layout', 'edgeGridLayout', edgeGridLayout);
Cytoscape.use(edgeGridLayout);

const NetworkGraph = ({
    nodes,
    networkFlowMapping,
    onNodeClick,
    onNamespaceClick,
    onClickOutside,
    filterState,
    setNetworkGraphRef,
    setSelectedNamespace
}) => {
    const [selectedNode, setSelectedNode] = useState();
    const [hoveredNode, setHoveredNode] = useState();
    const [firstRenderFinished, setFirstRenderFinished] = useState(false);
    const nodeSideMapRef = useRef({});
    const zoomFontMapRef = useRef({});
    const nodeSideMap = nodeSideMapRef.current;
    const zoomFontMap = zoomFontMapRef.current;
    const cy = useRef();
    const tippy = useRef();
    const namespacesWithDeployments = {};

    const data = nodes.map(datum => ({
        ...datum,
        isActive: filterState !== filterModes.active && datum.internetAccess
    }));

    const links = getLinks(data, networkFlowMapping);

    function makePopperDiv(text) {
        const div = document.createElement('div');
        div.classList.add('popper');
        div.innerHTML = text;
        document.body.appendChild(div);
        return div;
    }

    function getSideMap(source, target) {
        return nodeSideMap && nodeSideMap[source] && nodeSideMap[source][target]
            ? nodeSideMap[source][target]
            : null;
    }

    function createTippy(elm, text) {
        if (!elm) return;
        const popperRef = elm.popperRef();
        if (tippy.current) tippy.current.destroy();

        tippy.current = new Tippy(popperRef, {
            content: makePopperDiv(text),
            arrow: true,
            delay: 0,
            duration: 0
        });

        tippy.current.show();
    }

    function getNSEdges(nodeId) {
        const delimiter = '**__**';

        const filteredLinks = links.filter(
            ({ source, target, isActive, sourceNS, targetNS }) =>
                (!nodeId ||
                    source === nodeId ||
                    target === nodeId ||
                    sourceNS === nodeId ||
                    targetNS === nodeId) &&
                (filterState !== filterModes.active || isActive) &&
                sourceNS &&
                targetNS &&
                sourceNS !== targetNS
        );

        const sourceTargetMap = {};
        const activeLinkMap = filteredLinks.reduce((acc, curr) => {
            const { sourceNS, targetNS, isActive } = curr;
            const key = [sourceNS, targetNS].sort().join(delimiter);
            if (isActive) acc[key] = true;
            return acc;
        }, {});

        const counts = filteredLinks.reduce((acc, curr) => {
            const sourceTargetKey = [curr.source, curr.target].sort().join(delimiter);
            if (sourceTargetMap[sourceTargetKey]) {
                return acc;
            }

            sourceTargetMap[sourceTargetKey] = true;
            const key = [curr.sourceNS, curr.targetNS].sort().join(delimiter);
            acc[key] = acc[key] ? acc[key] + 1 : 1;
            return acc;
        }, {});

        return Object.keys(counts).map(key => {
            const [sourceId, targetId] = key.split(delimiter);
            const count = counts[key];
            const isActive = activeLinkMap[key];
            const activeClass = filterState !== filterModes.allowed && isActive ? 'active' : '';
            const { source, target, sourceSide } = getSideMap(sourceId, targetId) || {
                sourceId,
                targetId
            };
            const taxiDirClass = ['top', 'bottom'].includes(sourceSide)
                ? 'taxi-vertical'
                : 'taxi-horizontal';

            return {
                data: {
                    source,
                    target,
                    count
                },
                classes: `namespace ${activeClass} ${taxiDirClass}`
            };
        });
    }

    function getEdgesFromNode(nodeId, isNonIsolatedNode) {
        const edgeMap = {};
        const edges = [];
        if (isNonIsolatedNode && filterState !== filterModes.all) return edges;
        links.forEach(linkItem => {
            const { source, sourceNS, sourceName } = linkItem;
            const { target, targetNS, targetName, isActive } = linkItem;
            const nodeIsSource = nodeId === source;
            const nodeIsTarget = nodeId === target;
            // destination node info needed for network flow tab
            const destNodeId = nodeIsSource ? target : source;
            const destNodeNS = nodeIsSource ? targetNS : sourceNS;
            const destNodeName = nodeIsSource ? targetName : sourceName;
            if (
                (nodeIsSource || nodeIsTarget) &&
                (filterState !== filterModes.active || isActive)
            ) {
                const activeClass = filterState !== filterModes.allowed && isActive ? 'active' : '';
                const nonIsolatedClass = isNonIsolatedNode && !isActive ? 'nonIsolated' : '';
                const id = [source, target].sort().join('--');
                if (!edgeMap[id]) {
                    // If same namespace, draw line between the two nodes
                    if (sourceNS === targetNS) {
                        edges.push({
                            data: {
                                destNodeId,
                                destNodeNS,
                                destNodeName,
                                ...linkItem
                            },
                            classes: `edge ${activeClass} ${nonIsolatedClass}`
                        });
                    } else {
                        // make sure both nodes have edges drawn to the nearest side of their NS
                        let sourceNSSide = sourceNS;
                        let targetNSSide = targetNS;
                        const sideMap = getSideMap(sourceNS, targetNS);
                        if (sideMap) {
                            sourceNSSide = sideMap.source;
                            targetNSSide = sideMap.target;
                        }

                        // Edge from source to it's namespace
                        edges.push({
                            data: {
                                source,
                                target: sourceNSSide
                            },
                            classes: `edge inner ${activeClass} ${nonIsolatedClass}`
                        });

                        // Edge from target to its namespace
                        edges.push({
                            data: {
                                source: target,
                                target: targetNSSide,
                                destNodeId,
                                destNodeName,
                                destNodeNS,
                                isActive
                            },
                            classes: `edge inner ${activeClass} ${nonIsolatedClass}`
                        });
                    }
                    edgeMap[id] = true;
                }
            }
        });

        return edges;
    }

    function getDeploymentsList() {
        const filteredData = data.filter(datum => datum.entity && datum.entity.deployment);
        const deploymentList = filteredData.map(datum => {
            const { entity, ...datumProps } = datum;
            const { deployment, ...entityProps } = entity;
            const { namespace, ...deploymentProps } = deployment;
            const isSelected = !!(selectedNode && selectedNode.id === entity.id);
            const isHovered = !!(hoveredNode && hoveredNode.id === entity.id);
            const isBackground =
                !(selectedNode === undefined && hoveredNode === undefined) &&
                !isHovered &&
                !isSelected;
            const isNonIsolated = nonIsolated(datum);
            const classes = getClasses({
                active: datum.isActive,
                selected: isSelected,
                deployment: true,
                hovered: isHovered,
                background: isBackground,
                nonIsolated: isNonIsolated
            });

            const deploymentNode = {
                data: {
                    ...datumProps,
                    ...entityProps,
                    ...deploymentProps,
                    parent: namespace,
                    edges: getEdgesFromNode(entityProps.id),
                    deploymentId: entityProps.id
                },
                classes
            };
            return deploymentNode;
        });
        return deploymentList;
    }

    function getNodes() {
        const filteredData = data.filter(datum => datum.entity && datum.entity.deployment);
        const deploymentList = getDeploymentsList();
        const activeNamespaces = filteredData.reduce((acc, curr) => {
            const nsName = curr.entity.deployment.namespace;
            if (
                deploymentList.find(
                    element => element.data.isActive && element.data.parent === nsName
                )
            ) {
                acc.push(nsName);
            }

            return acc;
        }, []);

        const namespaceList = uniq(
            filteredData.map(datum => datum.entity.deployment.namespace)
        ).map(namespace => {
            const active = activeNamespaces.includes(namespace);
            const isHovered =
                hoveredNode && (hoveredNode.id === namespace || hoveredNode.parent === namespace);
            const isSelected =
                selectedNode &&
                (selectedNode.id === namespace || selectedNode.parent === namespace);
            const isBackground =
                !(selectedNode === undefined && hoveredNode === undefined) &&
                !isHovered &&
                !isSelected;
            const classes = getClasses({
                nsActive: active,
                nsSelected: isSelected,
                nsHovered: isHovered,
                background: isBackground
            });

            return {
                data: {
                    id: namespace,
                    name: `${active ? '\ue901' : ''} ${namespace}`,
                    active
                },
                classes
            };
        });

        const namespaceEdgeNodes = namespaceList.reduce((acc, namespace) => {
            const nsName = namespace.data.id;
            const set = ['top', 'left', 'right', 'bottom'];

            const newNodes = set.map(side => ({
                data: {
                    id: `${nsName}_${side}`,
                    parent: nsName,
                    side
                },
                classes: 'nsEdge'
            }));
            return acc.concat(newNodes);
        }, []);

        namespaceList.forEach(namespace => {
            deploymentList.forEach(deployment => {
                if (!namespacesWithDeployments[namespace.data.id]) {
                    namespacesWithDeployments[namespace.data.id] = [];
                }
                if (deployment.data.parent === namespace.data.id) {
                    namespacesWithDeployments[namespace.data.id].push(deployment);
                }
            });
        });

        return [...namespaceList, ...deploymentList, ...namespaceEdgeNodes];
    }

    function getEdges() {
        const node = hoveredNode || selectedNode;
        let allEdges = getNSEdges(node && node.id);
        if (node) {
            allEdges = allEdges.concat(getEdgesFromNode(node.id, nonIsolated(node)));
        }
        return allEdges;
    }

    function nodeHoverHandler(ev) {
        const node = ev.target.data();
        const { id, name, parent, side } = node;
        const isChild = !!parent;
        if (!cy || !isChild || side) return;

        setHoveredNode(node);
        const nodeElm = cy.current.getElementById(id);
        const parentElm = cy.current.getElementById(parent);
        createTippy(nodeElm, name);
        const children = parentElm.descendants();
        children.removeClass('background');
    }

    function nodeMouseOutHandler() {
        setHoveredNode();
    }

    function getNodeData(id) {
        return getDeploymentsList().filter(node => node.data.deploymentId === id);
    }

    function clickHandler(ev) {
        const { target } = ev;
        const evData = target.data && target.data();
        const id = evData && evData.id;
        const isParent = target.isParent && target.isParent();
        const isEdge = target.isEdge && target.isEdge();

        // Canvas or Selected node click: clear selection
        if (!evData || (selectedNode && evData && id === selectedNode.id)) {
            setSelectedNode();
            onClickOutside();
            return;
        }

        // Edge click or edge node click
        if (isEdge || evData.side) return;

        // Parent Click
        if (isParent) {
            if (id) {
                onNamespaceClick({ id, deployments: namespacesWithDeployments[id] || [] });
                setSelectedNode(evData);
            }
            return;
        }

        // Node click: select node
        if (target.isNode()) {
            setSelectedNode(evData);
            onNodeClick(evData);
        }

        if (!isParent) {
            setSelectedNamespace(null);
        }
    }

    function zoomHandler() {
        if (!cy || !cy.current) return;

        // to dynamically set the font size of namespace labels
        const zoomConstant = 20;
        const curZoomLevel = Math.round(cy.current.zoom() * zoomConstant);
        if (!zoomFontMap[curZoomLevel]) {
            zoomFontMap[curZoomLevel] = Math.max(
                (NS_FONT_SIZE / curZoomLevel) * zoomConstant,
                NS_FONT_SIZE
            );
        }
        cy.current.nodes(':parent').style('font-size', zoomFontMap[curZoomLevel]);
        cy.current.edges('.namespace').style('font-size', zoomFontMap[curZoomLevel]);
    }

    function zoomToFit() {
        if (!cy) return;
        cy.current.fit(null, GRAPH_PADDING);
        const newMinZoom = Math.min(cy.current.zoom(), cy.current.minZoom());
        cy.current.minZoom(newMinZoom);
        zoomHandler();
    }

    function zoomIn() {
        if (!cy.current) return;

        cy.current.zoom({
            level: Math.max(cy.current.zoom() + ZOOM_STEP, cy.current.minZoom())
        });
        cy.current.center();
    }

    function zoomOut() {
        if (!cy.current) return;

        cy.current.zoom({
            level: Math.min(cy.current.zoom() - ZOOM_STEP, MAX_ZOOM)
        });
        cy.current.center();
    }

    function getElements() {
        return { nodes: getNodes(), edges: getEdges() };
    }

    // Calculate which namespace box side combinations are shortest and store them
    function calculateNodeSideMap(changedNodeId) {
        if (!cy.current) return;

        // Get a map of all the side nodes per namespace
        const namespaces = cy.current.nodes(':parent');
        const sideNodesPerParent = namespaces.reduce((acc, namespace) => {
            const { id } = namespace.data(); // to

            const sideNodes = cy.current.nodes(`[parent="${id}"][side]`);

            const nodesInfo = sideNodes.map(node => {
                const { x, y } = node.position();
                const { side } = node.data();
                return {
                    node,
                    side,
                    id: node.id(),
                    x,
                    y
                };
            });
            return { ...acc, [id]: nodesInfo };
        }, {});

        const distances = {};

        function getDistance(sourceSideNode, targetSideNode) {
            const key = [sourceSideNode.id, targetSideNode.id].sort().join('**__**');
            const cachedDistance = distances[key];
            if (cachedDistance) return cachedDistance;
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

                if (i === j || (changedNodeId && ![sourceName, targetName].includes(changedNodeId)))
                    return;

                const targetSideNodes = sideNodesPerParent[targetName];
                let shortest;
                // check distances between every combination of side nodes to find shortest
                sourceSideNodes.forEach(sourceSideNode => {
                    const sourceSide = sourceSideNode.side;
                    const targetSideNode = targetSideNodes.find(tgtNode => {
                        const { side } = tgtNode;
                        if (sourceSide === 'top') return side === 'bottom';
                        if (sourceSide === 'bottom') return side === 'top';
                        if (sourceSide === 'left') return side === 'right';
                        if (sourceSide === 'right') return side === 'left';
                        return false;
                    });

                    const distance = getDistance(sourceSideNode, targetSideNode);
                    if (!shortest || shortest.distance > distance) {
                        shortest = {
                            source: sourceSideNode.id,
                            target: targetSideNode.id,
                            sourceSide: sourceSideNode.side,
                            targetSide: targetSideNode.side,
                            distance
                        };
                    }
                });
                sourceMap[targetName] = shortest;
            });
        });
    }

    function handleDrag(ev) {
        let changedNodeId;
        if (ev && ev.target) changedNodeId = ev.target.data().id;

        calculateNodeSideMap(changedNodeId);
        const newEdges = getEdges();

        cy.current.remove('edge');
        cy.current.add(newEdges);
    }

    function configureCY(cyInstance) {
        cy.current = cyInstance;
        cy.current
            .off('click mouseover mouseout mousedown drag')
            .on('click', clickHandler)
            .on('mouseover', 'node', throttle(nodeHoverHandler, 100))
            .on('mouseout', 'node', nodeMouseOutHandler)
            .on('mouseout mousedown', 'node', () => {
                if (tippy.current) tippy.current.destroy();
            })
            .on('drag', throttle(handleDrag, 100))
            .on('zoom', zoomHandler)
            .ready(() => {
                if (firstRenderFinished) return;
                setFirstRenderFinished(true);
                zoomToFit();
            });
    }

    const elements = getElements();
    // Effects
    function setWindowResize() {
        window.addEventListener('resize', throttle(() => zoomToFit, 100));

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
            getNodeData,
            onNodeClick
        });
    }

    function runLayout() {
        if (!cy.current) return;
        const CY = cy.current;
        const NSPositions = getParentPositions(CY.nodes(), { x: 100, y: 100 }); // all nodes, padding

        NSPositions.forEach(position => {
            const { id, x, y } = position;
            CY.layout({
                name: 'edgeGridLayout',
                parentPadding: { bottom: 5, top: 0, left: 0, right: 0 },
                position: { x, y },
                eles: CY.nodes(`[parent="${id}"]`)
            }).run();
        });
        CY.fit(null, GRAPH_PADDING);
    }

    function grabifyNamespaces() {
        if (!cy.current) return;
        const CY = cy.current;
        CY.nodes(`[parent]`).ungrabify();
    }

    useEffect(setWindowResize, []);
    useEffect(setGraphRef, []);
    useEffect(runLayout, [nodes.length]);
    useEffect(grabifyNamespaces);
    useEffect(calculateNodeSideMap);

    const normalizedElements = CytoscapeComponent.normalizeElements(elements);

    const loader = !normalizedElements && (
        <div className="absolute flex h-full items-center justify-center pin-t w-full pointer-events-none">
            <GraphLoader isLoading />
        </div>
    );

    return (
        <div className="h-full w-full relative">
            <div id="cytoscapeContainer" className="w-full h-full">
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
    nodes: PropTypes.arrayOf(
        PropTypes.shape({
            entity: PropTypes.shape({
                type: PropTypes.string.isRequired,
                id: PropTypes.string.isRequired,
                deployment: PropTypes.shape({
                    name: PropTypes.string.isRequired
                })
            }).isRequired
        })
    ).isRequired,
    networkFlowMapping: PropTypes.shape({}).isRequired,
    onNamespaceClick: PropTypes.func.isRequired,
    onNodeClick: PropTypes.func.isRequired,
    onClickOutside: PropTypes.func.isRequired,
    filterState: PropTypes.number.isRequired,
    setNetworkGraphRef: PropTypes.func.isRequired,
    setSelectedNamespace: PropTypes.func.isRequired
};

const mapDispatchToProps = {
    setNetworkGraphRef: graphActions.setNetworkGraphRef,
    setSelectedNamespace: graphActions.setSelectedNamespace
};

export default connect(
    null,
    mapDispatchToProps
)(NetworkGraph);
