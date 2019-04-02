import React, { useState, useRef, useEffect } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { actions as graphActions } from 'reducers/network/graph';

import Cytoscape from 'cytoscape';
import CytoscapeComponent from 'react-cytoscapejs';
import coseBilkentPlugin from 'cytoscape-cose-bilkent';
import popper from 'cytoscape-popper';
import Tippy from 'tippy.js';
import { uniq, debounce } from 'lodash';

import { coseBilkent as layout } from 'Containers/Network/Graph/networkGraphLayouts';
import filterModes from 'Containers/Network/Graph/filterModes';
import style from 'Containers/Network/Graph/networkGraphStyles';
import { getLinks, nonIsolated } from 'utils/networkGraphUtils';
import { MAX_ZOOM, MIN_ZOOM, ZOOM_STEP, GRAPH_PADDING } from 'constants/cytoscapeGraph';

Cytoscape.use(coseBilkentPlugin);
Cytoscape.use(popper);

function getClasses(map) {
    return Object.entries(map)
        .filter(entry => entry[1])
        .map(entry => entry[0])
        .join(' ');
}
let timeStamp = 0;
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
    const cy = useRef();
    const tippy = useRef();
    const namespacesWithDeployments = {};

    const data = nodes.map(datum => ({
        ...datum,
        isActive: filterState !== filterModes.active && datum.internetAccess
    }));

    function makePopperDiv(text) {
        const div = document.createElement('div');
        div.classList.add('popper');
        div.innerHTML = text;
        document.body.appendChild(div);
        return div;
    }

    // function createEdgePopper(elm, text) {
    //     const popperElm = elm.popper({
    //         content: makePopperDiv(text),
    //         popper: {
    //             removeOnDestroy: true
    //         }
    //     });
    //     const updatePopper = () => popperElm.scheduleUpdate();

    //     elm.connectedNodes().on('position', updatePopper);
    //     elm.connectedNodes()
    //         .parent()
    //         .on('position', updatePopper);
    //     cy.on('pan zoom resize', updatePopper);
    // }

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

    const namespaceMap = {};
    function getNamespace(nodeId) {
        if (!nodeId) return null;

        if (namespaceMap[nodeId]) return namespaceMap[nodeId];

        const match = data.find(datum => {
            if (datum && datum.entity) return datum.entity.id === nodeId;
            return false;
        });

        if (!match || !match.entity.deployment) return null;

        namespaceMap[nodeId] = match.entity.deployment.namespace;
        return match.entity.deployment.namespace;
    }

    function getEdgesFromNode(nodeId) {
        const links = getLinks(data, networkFlowMapping);

        const edgeMap = {};
        const allEdges = [];

        // Get all edges
        links.forEach(linkItem => {
            const { source, target, isActive } = linkItem;
            let modifiedLinkItem = null;
            if (
                (!nodeId || source === nodeId || target === nodeId) &&
                (filterState !== filterModes.active || isActive)
            ) {
                let sourceNS = getNamespace(source);
                let targetNS = getNamespace(target);
                const temp = sourceNS;
                if (source === nodeId) {
                    modifiedLinkItem = Object.assign({}, linkItem);
                }

                if (target === nodeId) {
                    sourceNS = targetNS;
                    targetNS = temp;
                    modifiedLinkItem = Object.assign(
                        {},
                        {
                            source: nodeId,
                            target: source,
                            targetName: linkItem.sourceName,
                            sourceName: linkItem.targetName
                        }
                    );
                }

                const edge = {
                    data: {
                        sourceNS,
                        targetNS,
                        ...modifiedLinkItem
                    },
                    classes: `node ${
                        filterState !== filterModes.allowed && isActive ? 'active' : ''
                    }`
                };
                const id = [source, target].sort().join('--');
                if (!edgeMap[id]) allEdges.push(edge);
                edgeMap[id] = true;
            }
        });

        // Get namespace counts
        const NSEdgeCounts = allEdges
            .filter(edge => edge.data.sourceNS !== edge.data.targetNS)
            .reduce((acc, curr) => {
                const { sourceNS, targetNS } = curr.data;
                const id = [sourceNS, targetNS].sort().join(',');
                if (!acc[id]) acc[id] = 1;
                else acc[id] += 1;
                return acc;
            }, {});

        // Create NS Edges with count
        const NSEdges = Object.keys(NSEdgeCounts).map(NSId => {
            const [source, target] = NSId.split(',');
            const count = NSEdgeCounts[NSId];
            const modifiedData = {
                source,
                target,
                count
            };

            allEdges.forEach(edge => {
                if (edge.data.source === nodeId) {
                    Object.assign(modifiedData, {
                        source: edge.data.sourceNS,
                        target: edge.data.targetNS,
                        targetNS: edge.data.targetNS,
                        targetId: edge.data.target,
                        targetName: edge.data.targetName
                    });
                }

                if (edge.data.target === nodeId) {
                    Object.assign(modifiedData, {
                        source: edge.data.targetNS,
                        target: edge.data.sourceNS,
                        targetId: edge.data.source,
                        targetNS: edge.data.sourceNS,
                        targetName: edge.data.sourceName
                    });
                }
            });
            return {
                data: modifiedData,
                classes: `namespace`
            };
        });

        // remove inter namespace edges
        const nodeEdges = allEdges.filter(edge => edge.data.sourceNS === edge.data.targetNS);
        return [...nodeEdges, ...NSEdges];
    }

    function getDeploymentsList() {
        const filteredData = data.filter(datum => datum.entity && datum.entity.deployment);
        const deploymentList = filteredData.map(datum => {
            const { entity, ...datumProps } = datum;
            const { deployment, ...entityProps } = entity;
            const { namespace: parent, ...deploymentProps } = deployment;
            const isSelected = selectedNode && selectedNode.id === entity.id;
            const isNonIsolated = nonIsolated(datum);
            const classes = getClasses({
                active: datum.isActive,
                selected: isSelected,
                nonIsolated: isNonIsolated
            });

            return {
                data: {
                    ...datumProps,
                    ...entityProps,
                    ...deploymentProps,
                    parent,
                    edges: getEdgesFromNode(entityProps.id),
                    deploymentId: entityProps.id
                },
                classes
            };
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
            const hoveredClassName = hoveredNode && hoveredNode.id === namespace ? 'nsHovered' : '';
            const namespaceClassName =
                selectedNode && selectedNode.parent === namespace ? 'nsSelected' : hoveredClassName;
            return {
                data: {
                    id: namespace,
                    name: `${active ? '\ue901' : ''} ${namespace}`,
                    active
                },
                classes: active ? 'nsActive' : namespaceClassName
            };
        });

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

        return [...namespaceList, ...deploymentList];
    }

    function getEdges() {
        if (hoveredNode || selectedNode) {
            const node = selectedNode || hoveredNode;
            return getEdgesFromNode(node.id);
        }

        return [];
    }

    function nodeHoverHandler(ev) {
        setHoveredNode(ev.target.data());
        const { name, parent, id } = ev.target.data();
        const isChild = !!parent;
        if (!cy || !isChild) return;
        const nodeElm = cy.current.getElementById(id);
        createTippy(nodeElm, name);
    }

    function nodeMouseOutHandler() {
        setHoveredNode();
    }

    function getNodeData(id) {
        return getDeploymentsList().filter(node => node.data.deploymentId === id);
    }

    function clickHandler(ev) {
        const currTimeStamp = new Date().getSeconds();
        // prevent handler from being called multiple times
        if (currTimeStamp - timeStamp === 0) {
            return;
        }
        timeStamp = currTimeStamp;
        // Canvas or Selected node click: clear selection
        if (
            !ev.target.data ||
            (selectedNode && ev.target.data() && ev.target.data().id === selectedNode.id)
        ) {
            setSelectedNode();
            onClickOutside();
            return;
        }

        // Parent Click: Do nothing
        if (ev.target.isParent()) {
            const { id } = ev.target.data();
            if (id) {
                onNamespaceClick({ id, deployments: namespacesWithDeployments[id] || [] });
                setSelectedNode();
            }
            return;
        }

        // Node click: select node
        const node = ev.target.data();
        setSelectedNode(node);
        onNodeClick(node);
        if (!ev.target.isParent()) {
            setSelectedNamespace(null);
        }
    }

    function zoomToFit() {
        if (!cy) return;
        cy.current.fit([], GRAPH_PADDING);
        const curFitZoom = cy.current.zoom();
        const curMinZoom = cy.current.minZoom();
        if (curFitZoom >= MIN_ZOOM) {
            let newMinZoom = curFitZoom;
            if (curMinZoom !== MIN_ZOOM) newMinZoom = Math.min(curFitZoom, curMinZoom);
            cy.current.minZoom(newMinZoom);
        }
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

    function configureCY(cyInstance) {
        cy.current = cyInstance;
        cy.current
            .on('click', null, clickHandler)
            .on('mouseover', 'node', debounce(nodeHoverHandler, 100))
            .on('mouseout', 'node', nodeMouseOutHandler)
            .on('mouseout mousedown', 'node', () => {
                if (tippy.current) tippy.current.destroy();
            })
            .ready(() => {
                if (firstRenderFinished) return;
                setFirstRenderFinished(true);
                zoomToFit();
            });
    }

    const elements = getElements();

    // Effects
    function handleWindowResize() {
        window.addEventListener('resize', debounce(() => zoomToFit, 100));

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
        cy.current.layout(layout).run();
    }

    useEffect(handleWindowResize, []);
    useEffect(setGraphRef, []);
    useEffect(runLayout, [nodes.length]);

    return (
        <div className="h-full w-full relative">
            <div id="cytoscapeContainer" className="w-full h-full">
                <CytoscapeComponent
                    elements={CytoscapeComponent.normalizeElements(elements)}
                    layout={layout}
                    stylesheet={style}
                    cy={configureCY}
                    minZoom={MIN_ZOOM}
                    maxZoom={MAX_ZOOM}
                    style={{ width: '100%', height: '100%' }}
                />
            </div>
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
