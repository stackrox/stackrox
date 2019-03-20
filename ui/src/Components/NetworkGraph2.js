import React, { useEffect, useRef } from 'react';
import PropTypes from 'prop-types';
import Cytoscape from 'cytoscape';
import coseBilkentPlugin from 'cytoscape-cose-bilkent';
import { getLinks } from 'utils/networkGraphUtils';
import { uniq, debounce } from 'lodash';
import { coseBilkent as layout } from 'Containers/Network/Graph/networkGraphLayouts';
import filterModes from 'Containers/Network/Graph/filterModes';
import style from 'Containers/Network/Graph/networkGraphStyles';

Cytoscape.use(coseBilkentPlugin);

const NetworkGraph = ({ nodes, networkFlowMapping, onNodeClick, updateKey, filterState }) => {
    const selectedNode = useRef();
    let cy = useRef();

    const data = nodes.map(datum => ({
        ...datum,
        isActive: filterState !== filterModes.active && datum.internetAccess
    }));

    function getEdges(nodeId) {
        const edges = getLinks(data, networkFlowMapping)
            .filter(linkItem => !nodeId || linkItem.source === nodeId || linkItem.target === nodeId) // filter by specific nodeId
            .filter(linkItem => filterState !== filterModes.active || linkItem.isActive)
            .map(linkItem => ({
                data: linkItem,
                classes: linkItem.isActive ? 'active' : ''
            }));

        // TODO: check for edges in different namespace and consolidate them into one line

        return edges;
    }

    function getNodes() {
        const filteredData = data.filter(datum => datum.entity && datum.entity.deployment);
        const deploymentList = filteredData
            .map(datum => {
                const { entity, ...datumProps } = datum;
                const { deployment, ...entityProps } = entity;
                const { namespace: parent, ...deploymentProps } = deployment;
                return {
                    data: {
                        ...datumProps,
                        ...entityProps,
                        ...deploymentProps,
                        parent,
                        deploymentId: entityProps.id
                    },
                    classes: datum.isActive ? 'active' : ''
                };
            })
            .filter(dep => !!dep);

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
        ).map(namespace => ({
            data: {
                id: namespace
            },
            classes: activeNamespaces.includes(namespace) ? 'nsActive' : ''
        }));

        return [...namespaceList, ...deploymentList];
    }

    function showNodeEdges(node) {
        cy.remove('edge');
        if (!node) return;

        const edges = getEdges(node.id, node.parent);
        cy.add(edges);
    }

    function nodeHoverHandler(ev) {
        if (selectedNode.current) return;
        showNodeEdges(ev.target.data(), cy);
    }

    function nodeMouseOutHandler() {
        if (selectedNode.current) return;
        showNodeEdges();
    }

    function highlightNode(node) {
        cy.nodes().removeClass('selected');
        if (node) cy.nodes(`#${node.id}`).addClass('selected');
    }

    function clickHandler(ev) {
        // Canvas or Selected node click: clear selection
        if (
            !ev.target.data ||
            (selectedNode.current &&
                ev.target.data() &&
                ev.target.data().id === selectedNode.current.id)
        ) {
            selectedNode.current = null;
            showNodeEdges();
            highlightNode();
            return;
        }

        // Parent Click: Do nothing
        if (ev.target.isParent()) {
            return;
        }

        // Node click: select node
        const node = ev.target.data();
        selectedNode.current = node;
        showNodeEdges(node);
        highlightNode(node);
        onNodeClick(node);
    }

    // New Nodes: Create new cytoscape instance
    useEffect(
        () => {
            cy = Cytoscape({
                container: document.getElementById('cytoscapeContainer'),
                layout,
                style,
                elements: getNodes(nodes)
            })
                .on('click', null, ev => {
                    clickHandler(ev);
                })
                .on('mouseover', 'node', debounce(nodeHoverHandler, 100))
                .on('mouseout', 'node', nodeMouseOutHandler);
            window.CY = cy;

            // handle resizing
            window.addEventListener(
                'resize',
                debounce(() => {
                    cy.fit(null, 50);
                }, 100)
            );
            // Return cleanup function
            const cleanup = () => {
                window.removeEventListener('resize');
            };
            return cleanup;
        },
        [nodes.length, filterState]
    );

    // Edges updated: Maybe do something
    useEffect(
        () => {
            // console.log('key updated', updateKey);
        },
        [updateKey]
    );

    return (
        <div className="h-full w-full relative">
            <div id="cytoscapeContainer" className="w-full h-full" />
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
    onNodeClick: PropTypes.func.isRequired,
    updateKey: PropTypes.number.isRequired,
    filterState: PropTypes.number.isRequired
};

export default NetworkGraph;
