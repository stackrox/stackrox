import uniq from 'lodash/uniq';

import entityTypes from 'constants/entityTypes';
import { networkTraffic, nodeTypes } from 'constants/networkGraph';
import { filterModes } from 'constants/networkFilterModes';

export const edgeTypes = {
    NAMESPACE_EDGE: 'NAMESPACE_EDGE',
    NODE_TO_NODE_EDGE: 'NODE_TO_NODE_EDGE',
    NODE_TO_NAMESPACE_EDGE: 'NODE_TO_NAMESPACE_EDGE',
};
const LINK_DELIMITER = '**__**';

export const getIsExternal = (type) =>
    type === nodeTypes.EXTERNAL_ENTITIES || type === nodeTypes.CIDR_BLOCK;

export const getIsNonIsolatedNode = (node) => node.nonIsolatedIngress && node.nonIsolatedEgress;

export const getIsDeploymentNode = (type) => type === entityTypes.DEPLOYMENT;

export const getIsNamespaceNode = (type) => type === entityTypes.NAMESPACE;

export const getIsExternalEntitiesNode = (type) => type === nodeTypes.EXTERNAL_ENTITIES;

export const getIsCIDRBlockNode = (type) => type === nodeTypes.CIDR_BLOCK;

export const getIsNamespaceEdge = (type) => type === edgeTypes.NAMESPACE_EDGE;

export const getIsNodeToNodeEdge = (type) => type === edgeTypes.NODE_TO_NODE_EDGE;

export const getIsNodeToNamespaceEdge = (type) => type === edgeTypes.NODE_TO_NAMESPACE_EDGE;

export const getIsNodeExternal = (id, nodes) => {
    const node = nodes.find((datum) => {
        return datum.entity.id === id;
    });
    if (!node) {
        throw Error(`Node with id of (${id}) does not exist`);
    }
    return (
        node.entity.type === nodeTypes.EXTERNAL_ENTITIES ||
        node.entity.type === nodeTypes.CIDR_BLOCK
    );
};

/**
 * Gets the namespace value for a node
 *
 * @param {!Object} node
 *
 * @returns {!string}
 *
 */
export function getNodeNamespace(node) {
    const { deployment, id, type } = node.entity;
    const isExternalEntitiesNode = getIsExternalEntitiesNode(type);
    const isCIDRBlockNode = getIsCIDRBlockNode(type);
    const isDeploymentNode = getIsDeploymentNode(type);
    // since external node's don't have a namespace, we'll utilize their "id"s instead
    if (isExternalEntitiesNode || isCIDRBlockNode) {
        return id;
    }
    if (isDeploymentNode) {
        return deployment.namespace;
    }
    throw new Error(`Node with unexpected type (${type}) was supplied to function`);
}

/**
 * Gets the name value for a node
 *
 * @param {!Object} node
 *
 * @returns {!string}
 *
 */
export function getNodeName(node) {
    const { deployment, type } = node.entity;
    const isExternalEntitiesNode = getIsExternalEntitiesNode(type);
    const isCIDRBlockNode = getIsCIDRBlockNode(type);
    const isDeploymentNode = getIsDeploymentNode(type);
    // since external node's don't have a unique name, we'll utilize their "id"s instead
    if (isExternalEntitiesNode) {
        return 'External Entities';
    }
    if (isCIDRBlockNode) {
        return `${node.entity?.externalSource?.cidr} / ${node.entity?.externalSource?.name}`;
    }
    if (isDeploymentNode) {
        return deployment.name;
    }
    throw new Error(`Node with unexpected type (${type}) was supplied to function`);
}

/**
 * Create a key using a source and target with a delimiter in between
 *
 * @param {!string} source a string representing the source node
 * @param {!string} target a string representing the target node
 * @returns {!string}
 *
 * ex: getSourceTargetKey("source", "target") => "source**__**target"
 */
export const getSourceTargetKey = (source, target) => {
    return [source, target].sort().join(LINK_DELIMITER);
};

/**
 * Gets the source and target from a node link key
 *
 * @param {!string} sourceTargetKey a string representing a key using a source and target
 * @returns {!String[]}
 *
 * ex: getSourceTargetFromKey("source**__**target") => ["source", "target"]
 */
export const getSourceTargetFromKey = (sourceTargetKey) => {
    return sourceTargetKey.split(LINK_DELIMITER);
};

/**
 * Checks against nodeSideMap to return the closest side of the NS that a deployment is positioned
 *
 * @param {!string} source source deployment
 * @param {!string} target target deployment
 * @param {!Object} nodeSideMap map of least distanced sides between source and target deployments
 * @returns {!Object}
 */
export const getSideMap = (source, target, nodeSideMap) => {
    return nodeSideMap?.[source]?.[target] ? nodeSideMap[source][target] : null;
};

/**
 * Iterates through a mapping of classes to boolean types to return a string of appended classes
 *
 * @param {!Object} map object containing className to boolean properties
 * @returns {!string}
 *
 * ex:
 *  input: map = {
 *      isActive: true,
 *      isUnidirectional: false
 *  }
 * output: 'isActive isUnidirectional'
 */
export const getClasses = (map) => {
    return Object.entries(map)
        .filter((entry) => entry[1])
        .map((entry) => entry[0])
        .join(' ');
};

/**
 * Creates a mapping of ports/protocols based on node links (source->target), and then
 * returns a closure to allow getting the ports/protocols of a specific source->target
 *
 * @param {!Object[]} nodes
 * @param {!String} highlightedNodeId
 * @param {!Object} networkNodeMap
 * @param {!number} filterState
 * @returns {!Object}
 *
 */
export const createPortsAndProtocolsSelector = (
    nodes,
    highlightedNodeId,
    networkNodeMap,
    filterState
) => {
    const linkPortsAndProtocols = {};

    // create a mapping of node edges -> ports and protocols
    nodes.forEach((sourceNode) => {
        const targetNodeIds = Object.keys(sourceNode?.outEdges || {});
        targetNodeIds.forEach((targetNodeId) => {
            if (!networkNodeMap?.[targetNodeId]) {
                return;
            }
            const { allowed, active } = networkNodeMap[targetNodeId];
            let targetNode = allowed;
            if (filterState === filterModes.active) {
                targetNode = active;
            }
            const { properties } = sourceNode.outEdges[targetNodeId];
            if (
                (sourceNode.entity.type === entityTypes.DEPLOYMENT ||
                    sourceNode.entity.type === nodeTypes.EXTERNAL_ENTITIES ||
                    sourceNode.entity.type === nodeTypes.CIDR_BLOCK) &&
                (targetNode?.entity?.type === entityTypes.DEPLOYMENT ||
                    targetNode?.entity?.type === nodeTypes.EXTERNAL_ENTITIES ||
                    targetNode?.entity?.type === nodeTypes.CIDR_BLOCK)
            ) {
                const nodeLinkKey = getSourceTargetKey(sourceNode.entity.id, targetNode.entity.id);
                const traffic =
                    targetNode.entity.id === highlightedNodeId
                        ? networkTraffic.INGRESS
                        : networkTraffic.EGRESS;
                const modifiedProperties =
                    properties?.map((datum) => {
                        return { ...datum, traffic };
                    }) || [];
                if (linkPortsAndProtocols[nodeLinkKey]) {
                    linkPortsAndProtocols[nodeLinkKey] = [
                        ...linkPortsAndProtocols[nodeLinkKey],
                        ...modifiedProperties,
                    ];
                } else {
                    linkPortsAndProtocols[nodeLinkKey] = [...modifiedProperties];
                }
            }
        });
    });

    function getPortsAndProtocolsByLink(nodeLinkKey, isEgress) {
        if (linkPortsAndProtocols[nodeLinkKey]) {
            return linkPortsAndProtocols[nodeLinkKey];
        }
        if (typeof isEgress !== 'boolean') {
            throw Error('The value for isEgress must be set');
        }
        // if the mapping doesn't contain the ports/protocols information, it's because we create
        // additional edges between egress non-isolated and ingress non-isolated nodes. For those cases,
        // we want to default to showing Any protocols/ Any ports
        const traffic = isEgress ? 'egress' : 'ingress';
        return [{ port: '*', protocol: 'L4_PROTOCOL_ANY', traffic }];
    }

    return getPortsAndProtocolsByLink;
};

function addSimulatedStatusToNamespaceLink(namespaceLink, simulatedStatus) {
    const newNamespaceLink = { ...namespaceLink };
    if (!namespaceLink.simulatedStatus) {
        newNamespaceLink.simulatedStatus = simulatedStatus;
    } else if (
        (namespaceLink.simulatedStatus === 'ADDED' && simulatedStatus !== 'ADDED') ||
        (namespaceLink.simulatedStatus === 'REMOVED' && simulatedStatus !== 'REMOVED') ||
        (namespaceLink.simulatedStatus === 'UNMODIFIED' && simulatedStatus !== 'UNMODIFIED')
    ) {
        newNamespaceLink.simulatedStatus = 'MODIFIED';
    }
    return newNamespaceLink;
}

/**
 * Iterates through a list of links and returns bundled edges between namespaces
 *
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains links, filterState, nodeSideMap
 * @returns {!Object[]} list of objects describing bundled edges between namespaces
 */
export const getNamespaceEdges = ({
    nodes = [],
    unfilteredLinks = [],
    filterState,
    nodeSideMap,
    selectedNode,
    hoveredNode,
    hoveredEdge,
    networkNodeMap,
}) => {
    const visitedNodeLinks = {};
    const activeNamespaceLinks = {};
    const namespaceLinks = {};
    const highlightedNodeId = (hoveredNode || selectedNode)?.id;
    const getPortsAndProtocolsByLink = createPortsAndProtocolsSelector(
        nodes,
        highlightedNodeId,
        networkNodeMap,
        filterState
    );

    const filteredLinks = unfilteredLinks.filter(
        ({ source, target, isActive, sourceNS, targetNS }) =>
            source &&
            target &&
            (!highlightedNodeId ||
                source === highlightedNodeId ||
                target === highlightedNodeId ||
                sourceNS === highlightedNodeId ||
                targetNS === highlightedNodeId) &&
            (filterState !== filterModes.active || isActive) &&
            sourceNS &&
            targetNS &&
            sourceNS !== targetNS
    );

    filteredLinks.forEach(
        ({
            source,
            target,
            sourceNS,
            targetNS,
            isActive,
            isAllowed,
            isExternal,
            simulatedStatus,
        }) => {
            const namespaceLinkKey = getSourceTargetKey(sourceNS, targetNS);
            const nodeLinkKey = getSourceTargetKey(source, target);
            const isEgress = source === highlightedNodeId;

            // keep track of which namespace links are active
            if (isActive) {
                activeNamespaceLinks[namespaceLinkKey] = true;
            }

            const portsAndProtocols = getPortsAndProtocolsByLink(nodeLinkKey, isEgress);
            const isLinkPreviouslyVisited = visitedNodeLinks[nodeLinkKey];

            let namespaceLink = namespaceLinks[namespaceLinkKey] || {
                portsAndProtocols: [],
                numBidirectionalLinks: 0,
                numUnidirectionalLinks: 0,
                numActiveBidirectionalLinks: 0,
                numActiveUnidirectionalLinks: 0,
                numAllowedBidirectionalLinks: 0,
                numAllowedUnidirectionalLinks: 0,
                isExternal,
            };

            // add simulated status information
            namespaceLink = addSimulatedStatusToNamespaceLink(namespaceLink, simulatedStatus);

            namespaceLink.portsAndProtocols = [
                ...namespaceLink.portsAndProtocols,
                ...portsAndProtocols,
            ];

            if (isLinkPreviouslyVisited) {
                namespaceLink.numBidirectionalLinks += 1;
                namespaceLink.numUnidirectionalLinks = namespaceLink.numUnidirectionalLinks
                    ? namespaceLink.numUnidirectionalLinks - 1
                    : 0;
                if (isActive) {
                    namespaceLink.numActiveBidirectionalLinks += 1;
                    namespaceLink.numActiveUnidirectionalLinks =
                        namespaceLink.numActiveUnidirectionalLinks
                            ? namespaceLink.numActiveUnidirectionalLinks - 1
                            : 0;
                }
                if (isAllowed) {
                    namespaceLink.numAllowedBidirectionalLinks += 1;
                    namespaceLink.numAllowedUnidirectionalLinks =
                        namespaceLink.numAllowedUnidirectionalLinks
                            ? namespaceLink.numAllowedUnidirectionalLinks - 1
                            : 0;
                }
            } else {
                namespaceLink.numUnidirectionalLinks += 1;
                if (isActive) {
                    namespaceLink.numActiveUnidirectionalLinks += 1;
                }
                if (isAllowed) {
                    namespaceLink.numAllowedUnidirectionalLinks += 1;
                }
                visitedNodeLinks[nodeLinkKey] = true;
            }

            namespaceLinks[namespaceLinkKey] = namespaceLink;
        }
    );

    const namespaceEdges = Object.keys(namespaceLinks).map((namespaceLinkKey) => {
        const [sourceNS, targetNS] = getSourceTargetFromKey(namespaceLinkKey);
        const {
            portsAndProtocols,
            numBidirectionalLinks,
            numUnidirectionalLinks,
            numActiveBidirectionalLinks,
            numActiveUnidirectionalLinks,
            numAllowedBidirectionalLinks,
            numAllowedUnidirectionalLinks,
            isExternal,
            simulatedStatus,
        } = namespaceLinks[namespaceLinkKey];
        const isHoveredEdge =
            (hoveredEdge?.sourceNodeNamespace === sourceNS &&
                hoveredEdge?.targetNodeNamespace === targetNS) ||
            (hoveredEdge?.targetNodeNamespace === sourceNS &&
                hoveredEdge?.sourceNodeNamespace === targetNS);

        const isNamespaceActive = activeNamespaceLinks[namespaceLinkKey];
        const isNamespaceEdgeActive = filterState !== filterModes.allowed && isNamespaceActive;
        // this is to show the directionality of the external entities/CIDR block edges
        const isExternalEdge = (hoveredNode || selectedNode) && isExternal;

        const classes = getClasses({
            namespace: true,
            active: isNamespaceEdgeActive,
            hovered: isHoveredEdge,
            unidirectional: numUnidirectionalLinks > 0,
            bidirectional: numBidirectionalLinks > 0,
            externalEdge: isExternalEdge,
            simulated: !!simulatedStatus,
            added: simulatedStatus === 'ADDED',
            removed: simulatedStatus === 'REMOVED',
            unmodified: simulatedStatus === 'UNMODIFIED',
            modified: simulatedStatus === 'MODIFIED',
        });

        const { source, target } = getSideMap(sourceNS, targetNS, nodeSideMap) || {
            source: sourceNS,
            target: targetNS,
        };

        return {
            data: {
                // not exactly sure how these got flipped, but just swapped them for now here
                // TODO: fix NS edge direction at source
                source: target,
                target: source,
                sourceNodeNamespace: targetNS,
                targetNodeNamespace: sourceNS,
                numBidirectionalLinks,
                numUnidirectionalLinks,
                numActiveBidirectionalLinks,
                numActiveUnidirectionalLinks,
                numAllowedBidirectionalLinks,
                numAllowedUnidirectionalLinks,
                count: numBidirectionalLinks + numUnidirectionalLinks,
                portsAndProtocols,
                type: edgeTypes.NAMESPACE_EDGE,
            },
            classes,
        };
    });
    return namespaceEdges;
};

const getLinkMetadata = (link, isSourceNode) => {
    const {
        source: sourceNodeId,
        sourceNS: sourceNodeNamespace,
        sourceName: sourceNodeName,
        sourceType,
        target: targetNodeId,
        targetNS: targetNodeNamespace,
        targetName: targetNodeName,
        targetType,
        isActive,
        isAllowed,
        simulatedStatus,
    } = link;
    // destination node info needed for network flow tab
    const destNodeId = isSourceNode ? targetNodeId : sourceNodeId;
    const destNodeName = isSourceNode ? targetNodeName : sourceNodeName;
    const destNodeNamespace = isSourceNode ? targetNodeNamespace : sourceNodeNamespace;
    const destNodeType = isSourceNode ? targetType : sourceType;
    return {
        destNodeId,
        destNodeNamespace,
        destNodeName,
        destNodeType,
        sourceNodeId,
        sourceNodeName,
        sourceNodeNamespace,
        targetNodeId,
        targetNodeName,
        targetNodeNamespace,
        isActive,
        isAllowed,
        simulatedStatus,
    };
};

const setBidirectionalLinkData = (nodeLink, classes) => {
    const processedNodeLink = { ...nodeLink };
    // if this edge is already in the nodeLinks, it means it's going in the other direction
    processedNodeLink.data.isBidirectional = true;
    processedNodeLink.data.traffic = networkTraffic.BIDIRECTIONAL;
    processedNodeLink.classes = getClasses({
        ...classes,
        bidirectional: true,
    });
    return processedNodeLink;
};

const getIsInnerNamespaceEdge = (hoveredEdge, sourceNS, targetNS) => {
    // if the hovered edge is a namespace edge, it highlights all the edges connected to the namespaces
    const isOriginalDirectionHovered =
        hoveredEdge?.sourceNodeNamespace === sourceNS &&
        hoveredEdge?.targetNodeNamespace === targetNS;
    const isOppositeDirectionHovered =
        hoveredEdge?.sourceNodeNamespace === targetNS &&
        hoveredEdge?.targetNodeNamespace === sourceNS;
    return (
        getIsNamespaceEdge(hoveredEdge?.type) &&
        (isOriginalDirectionHovered || isOppositeDirectionHovered)
    );
};

const addSimulationClasses = (classes, simulatedStatus) => {
    if (!simulatedStatus) {
        return {};
    }
    if (
        classes &&
        ((classes.includes('added') && simulatedStatus !== 'ADDED') ||
            (classes.includes('removed') && simulatedStatus !== 'REMOVED') ||
            (classes.includes('unmodified') && simulatedStatus !== 'UNMODIFIED') ||
            (classes.includes('modified') && simulatedStatus !== 'MODIFIED'))
    ) {
        return {
            simulated: !!simulatedStatus,
            modified: true,
        };
    }
    const simulationClassName = simulatedStatus.toLowerCase();
    return {
        simulated: !!simulatedStatus,
        [simulationClassName]: true,
    };
};

/**
 * Iterates through links to return edges that are connected to a node
 *
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains links, filterState, and nodeSideMap
 * @returns {!Object[]}
 */
export const getEdgesFromNode = ({
    filterState,
    unfilteredLinks,
    nodes,
    nodeSideMap,
    hoveredNode,
    selectedNode,
    hoveredEdge,
    networkNodeMap,
}) => {
    // to prevent rerendering of duplicate edges
    const nodeLinks = {};
    const inAllowedFilterState = filterState === filterModes.allowed;
    const highlightedNode = hoveredNode || selectedNode;

    // if a node wasn't selected or hovered over, we don't want to show it's links
    if (!highlightedNode) {
        return [];
    }

    const getPortsAndProtocolsByLink = createPortsAndProtocolsSelector(
        nodes,
        highlightedNode?.id,
        networkNodeMap,
        filterState
    );

    unfilteredLinks.forEach((link) => {
        const {
            source,
            sourceNS,
            target,
            targetNS,
            isActive,
            isBetweenNonIsolated,
            simulatedStatus,
        } = link;
        const isSourceNode = highlightedNode?.id === source;
        const isTargetNode = highlightedNode?.id === target;

        const isEgress = isSourceNode;
        // if the currently hovered/selected node is a target for this link (ingress)
        const isRelativeIngress = isTargetNode;
        // if the currently hovered/selected node is a source for this link (egress)
        const isRelativeEgress = isSourceNode;
        const traffic = isRelativeIngress ? networkTraffic.INGRESS : networkTraffic.EGRESS;

        // to get target/source/destNode information
        const linkMetadata = getLinkMetadata(link, isSourceNode);

        // only get edges for the currently highlighted node
        if ((isSourceNode || isTargetNode) && (filterState !== filterModes.active || isActive)) {
            // if ingress or egress, show edge arrow to indicate direction
            const isUnidirectional = isRelativeIngress || isRelativeEgress;

            const isSourceExternal = getIsNodeExternal(source, nodes);
            const isTargetExternal = getIsNodeExternal(target, nodes);

            // getting category of link
            const linkIsInSameNamespace = sourceNS === targetNS;
            const linkIsInBetweenNamespaces = !isSourceExternal && !isTargetExternal;
            const linkIsExternal = isSourceExternal || isTargetExternal;

            // to access link in nodeLinks map
            const nodeLinkKey = getSourceTargetKey(source, target);
            const nodeLink = nodeLinks[nodeLinkKey];
            const portsAndProtocols = getPortsAndProtocolsByLink(nodeLinkKey, isEgress);

            const simulationClasses = addSimulationClasses(nodeLink?.classes, simulatedStatus);
            const coreClasses = {
                edge: true,
                active: !inAllowedFilterState && isActive,
                // only hide edge when it's bw nonisolated and is not active
                nonIsolated: isBetweenNonIsolated && (!isActive || inAllowedFilterState),
            };

            // if the edge is between two deployments in the same namespace
            if (linkIsInSameNamespace) {
                if (!nodeLink) {
                    const classes = getClasses({
                        ...coreClasses,
                        ...simulationClasses,
                        unidirectional: isUnidirectional,
                        // if the edge is in the same namespace, it's hovered when the source/target lines up
                        hovered:
                            hoveredEdge?.sourceNodeId === source &&
                            hoveredEdge?.targetNodeId === target,
                    });
                    nodeLinks[nodeLinkKey] = {
                        data: {
                            ...linkMetadata,
                            portsAndProtocols,
                            traffic,
                            type: edgeTypes.NODE_TO_NODE_EDGE,
                            ...link,
                        },
                        classes,
                    };
                } else if (!nodeLink?.data?.isBidirectional) {
                    // if this edge is already in the nodeLinks, it means it's going in the other direction
                    const classes = {
                        ...coreClasses,
                        ...simulationClasses,
                        // if the edge is bidirectional, it means the source/target is backwards if hovered
                        hovered:
                            hoveredEdge?.targetNodeId === source &&
                            hoveredEdge?.sourceNodeId === target,
                    };
                    nodeLinks[nodeLinkKey] = setBidirectionalLinkData(nodeLink, classes);
                }
            } else {
                // get inner edge for both source and target nodes
                let sourceParentSide = isSourceExternal ? source : sourceNS;
                let targetParentSide = isTargetExternal ? target : targetNS;
                const sideMap = getSideMap(sourceParentSide, targetParentSide, nodeSideMap);
                if (sideMap) {
                    sourceParentSide = sideMap.source;
                    targetParentSide = sideMap.target;
                }

                // determine whether highlighted node (hovered or selected) is within the same NS as source/target
                const highlightedNodeInSourceNS = highlightedNode?.parent === sourceNS;
                const highlightedNodeInTargetNS = highlightedNode?.parent === targetNS;

                // if the hovered edge is a namespace edge, it highlights all the edges connected to the namespaces
                const isInnerNamespaceEdge = getIsInnerNamespaceEdge(
                    hoveredEdge,
                    sourceNS,
                    targetNS
                );

                // if an inner edge is to the source namespace side, it's hovered when the source is the same
                const isInnerSourceEdgeHovered =
                    isInnerNamespaceEdge ||
                    (getIsNodeToNamespaceEdge(hoveredEdge?.type) &&
                        (hoveredEdge?.sourceNodeId === source ||
                            hoveredEdge?.targetNodeId === source));
                // if an inner edge is to the target namespace side, it's hovered when the target is the same
                const isInnerTargetEdgeHovered =
                    isInnerNamespaceEdge ||
                    (getIsNodeToNamespaceEdge(hoveredEdge?.type) &&
                        (hoveredEdge?.targetNodeId === target ||
                            hoveredEdge?.sourceNodeId === target));

                // getting inner edges from nodeLink map
                const innerSourceEdgeKey = getSourceTargetKey(source, sourceParentSide);
                const innerTargetEdgeKey = getSourceTargetKey(targetParentSide, target);
                const innerSourceEdge = nodeLinks[innerSourceEdgeKey];
                const innerTargetEdge = nodeLinks[innerTargetEdgeKey];

                const innerSourceEdgeSimulationClasses = addSimulationClasses(
                    innerSourceEdge?.classes,
                    simulatedStatus
                );
                const innerTargetEdgeSimulationClasses = addSimulationClasses(
                    innerTargetEdge?.classes,
                    simulatedStatus
                );

                // if the inner edge from source/target to namespace is in the same namespace as selected
                const innerSourceEdgeClasses = getClasses({
                    ...coreClasses,
                    ...innerSourceEdgeSimulationClasses,
                    unidirectional: isUnidirectional,
                    inner: true,
                    withinNS: highlightedNodeInSourceNS,
                    hovered: isInnerSourceEdgeHovered,
                });
                // Edge from source deployment to it's namespace edge
                const constructedInnerSourceEdge = {
                    data: {
                        source,
                        target: sourceParentSide,
                        ...linkMetadata,
                        portsAndProtocols,
                        type: edgeTypes.NODE_TO_NAMESPACE_EDGE,
                        traffic,
                    },
                    classes: innerSourceEdgeClasses,
                };

                const innerTargetEdgeClasses = getClasses({
                    ...coreClasses,
                    ...innerTargetEdgeSimulationClasses,
                    unidirectional: isUnidirectional,
                    inner: true,
                    withinNS: highlightedNodeInTargetNS,
                    hovered: isInnerTargetEdgeHovered,
                });
                // Edge from namespace edge to target deployment
                const constructedInnerTargetEdge = {
                    data: {
                        source: targetParentSide,
                        target,
                        ...linkMetadata,
                        portsAndProtocols,
                        type: edgeTypes.NODE_TO_NAMESPACE_EDGE,
                        traffic,
                    },
                    classes: innerTargetEdgeClasses,
                };

                if (linkIsInBetweenNamespaces) {
                    // else if the edge is between two different namespaces

                    // if the inner source edge does not exist in the link map yet, add it to the node link map
                    if (!innerSourceEdge) {
                        nodeLinks[innerSourceEdgeKey] = constructedInnerSourceEdge;
                    } else if (!innerSourceEdge.data?.isBidirectional) {
                        const classes = {
                            ...coreClasses,
                            ...innerSourceEdgeSimulationClasses,
                            inner: true,
                            withinNS: highlightedNodeInSourceNS,
                            hovered: isInnerSourceEdgeHovered,
                        };
                        nodeLinks[innerSourceEdgeKey] = setBidirectionalLinkData(
                            nodeLinks[innerSourceEdgeKey],
                            classes
                        );
                    }

                    // if the inner target edge does not exist yet, add it to the node link map
                    if (!innerTargetEdge) {
                        nodeLinks[innerTargetEdgeKey] = constructedInnerTargetEdge;
                    }
                } else if (linkIsExternal) {
                    const innerEdgesExist = !!(innerSourceEdge && innerTargetEdge);

                    // accounting for inner edges connected to external sources:
                    // if either the source or target inner edge does not exist and the source or target is external
                    if (!innerEdgesExist) {
                        // if the source is external, there must be an inner edge on the target side
                        if (isSourceExternal) {
                            if (!innerTargetEdge) {
                                nodeLinks[innerTargetEdgeKey] = constructedInnerTargetEdge;
                            } else if (!innerTargetEdge.data?.isBidirectional) {
                                const classes = {
                                    ...coreClasses,
                                    ...innerTargetEdgeSimulationClasses,
                                    inner: true,
                                    withinNS: highlightedNodeInTargetNS,
                                    hovered: isInnerTargetEdgeHovered,
                                };
                                nodeLinks[innerTargetEdgeKey] = setBidirectionalLinkData(
                                    nodeLinks[innerTargetEdgeKey],
                                    classes
                                );
                            }
                            if (!innerSourceEdge) {
                                const classes = getClasses({
                                    edge: true,
                                    inner: true,
                                    hidden: true,
                                    ...innerSourceEdgeSimulationClasses,
                                });
                                // Edge from external source to external source edge
                                nodeLinks[innerSourceEdgeKey] = {
                                    data: {
                                        source,
                                        target: sourceParentSide,
                                        ...linkMetadata,
                                        portsAndProtocols,
                                        type: edgeTypes.NODE_TO_NAMESPACE_EDGE,
                                        traffic,
                                    },
                                    classes,
                                };
                            }
                        }
                        // if the target is external, there must be an inner edge on the source side
                        else if (isTargetExternal) {
                            if (!innerSourceEdge) {
                                nodeLinks[innerSourceEdgeKey] = constructedInnerSourceEdge;
                            } else if (!innerSourceEdge.data?.isBidirectional) {
                                const classes = {
                                    ...coreClasses,
                                    ...innerSourceEdgeSimulationClasses,
                                    inner: true,
                                    withinNS: highlightedNodeInSourceNS,
                                    hovered: isInnerSourceEdgeHovered,
                                };
                                nodeLinks[innerSourceEdgeKey] = setBidirectionalLinkData(
                                    nodeLinks[innerSourceEdgeKey],
                                    classes
                                );
                            }
                            if (!innerTargetEdge) {
                                const classes = getClasses({
                                    edge: true,
                                    inner: true,
                                    hidden: true,
                                    ...innerTargetEdgeSimulationClasses,
                                });
                                // Edge from external source to external source edge
                                nodeLinks[innerTargetEdgeKey] = {
                                    data: {
                                        source: targetParentSide,
                                        target,
                                        ...linkMetadata,
                                        portsAndProtocols,
                                        type: edgeTypes.NODE_TO_NAMESPACE_EDGE,
                                        traffic,
                                    },
                                    classes,
                                };
                            }
                        }
                    }

                    if (
                        innerEdgesExist &&
                        !innerSourceEdge?.data?.isBidirectional &&
                        !highlightedNodeInSourceNS
                    ) {
                        if (!isSourceExternal) {
                            // if this edge is already in the nodeLinks, it means it's going in the other direction
                            const classes = {
                                ...coreClasses,
                                ...innerSourceEdgeSimulationClasses,
                                hovered: isInnerSourceEdgeHovered,
                            };
                            nodeLinks[innerSourceEdgeKey] = setBidirectionalLinkData(
                                nodeLinks[innerSourceEdgeKey],
                                classes
                            );
                        }

                        if (!isTargetExternal) {
                            // we want to make sure the corresponding inner edge from the other namespace is also updated
                            const classes = {
                                ...coreClasses,
                                ...innerTargetEdgeSimulationClasses,
                                hovered: isInnerTargetEdgeHovered,
                            };
                            nodeLinks[innerTargetEdgeKey] = setBidirectionalLinkData(
                                nodeLinks[innerTargetEdgeKey],
                                classes
                            );
                        }
                    }

                    if (
                        innerEdgesExist &&
                        !innerTargetEdge?.data?.isBidirectional &&
                        !highlightedNodeInTargetNS
                    ) {
                        if (!isTargetExternal) {
                            // if this edge is already in the nodeLinks, it means it's going in the other direction
                            const classes = {
                                ...coreClasses,
                                ...innerTargetEdgeSimulationClasses,
                                hovered: isInnerTargetEdgeHovered,
                            };
                            nodeLinks[innerTargetEdgeKey] = setBidirectionalLinkData(
                                nodeLinks[innerTargetEdgeKey],
                                classes
                            );
                        }

                        if (!isSourceExternal) {
                            // we want to make sure the corresponding inner edge from the other namespace is also updated
                            const classes = {
                                ...coreClasses,
                                ...innerSourceEdgeSimulationClasses,
                                hovered: isInnerSourceEdgeHovered,
                            };
                            nodeLinks[innerSourceEdgeKey] = setBidirectionalLinkData(
                                nodeLinks[innerSourceEdgeKey],
                                classes
                            );
                        }
                    }
                }
            }
        }
    });

    return Object.values(nodeLinks);
};

function getAdjacentNodeIdsToNode(node, filterState) {
    if (filterState === filterModes.active) {
        const egressIds = node?.egress || [];
        const ingressIds = node?.ingress || [];
        return [...egressIds, ...ingressIds];
    }
    return node?.edges?.reduce((acc, curr) => [...acc, curr.data.destNodeId], []);
}

function getIsAdjacent(node, entityId, filterState) {
    const adjacentNodeIdsToNode = getAdjacentNodeIdsToNode(node, filterState);
    return !!adjacentNodeIdsToNode?.find((id) => entityId === id);
}

// to determine whether to dim or highlight the current node based on adjacency within the graph
export const getIsAdjacentToHighlightedNode = ({
    selectedNode,
    hoveredNode,
    entityId,
    filterState,
}) => {
    const isSelected = !!(selectedNode?.id === entityId);
    const isHovered = !!(hoveredNode?.id === entityId);
    const isAdjacentToSelected = selectedNode && getIsAdjacent(selectedNode, entityId, filterState);
    const isAdjacentToHovered = hoveredNode && getIsAdjacent(hoveredNode, entityId, filterState);
    return (!isHovered && isAdjacentToHovered) || (!isSelected && isAdjacentToSelected);
};

export const getDirectionalityEdges = (node, filterState) => {
    let ingress = [];
    let egress = [];
    if (node) {
        const {
            ingressAllowed = [],
            ingressActive = [],
            egressAllowed = [],
            egressActive = [],
        } = node;
        if (filterState === filterModes.allowed) {
            ingress = ingressAllowed;
            egress = egressAllowed;
        } else if (filterState === filterModes.active) {
            ingress = ingressActive;
            egress = egressActive;
        } else {
            ingress = [...ingressActive, ...ingressAllowed];
            egress = [...egressActive, ...egressAllowed];
        }
    }
    return { ingress, egress };
};

/**
 * Iterates through a list of nodes to return a list of deployments with proper styling classes for cytoscape
 *
 * @param {!Object[]} filteredData list of deployments
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains links, filterState, and nodeSideMap,
 *                            networkNodeMap, hoveredNode, and selectedNode
 * @returns {!Object[]}
 */
export const getDeploymentList = (filteredData, configObj = {}) => {
    const { hoveredNode, selectedNode, filterState, networkNodeMap } = configObj;
    const deploymentList = filteredData.map((datum) => {
        const { entity, ...datumProps } = datum;
        const { deployment, ...entityProps } = entity;
        const { namespace, ...deploymentProps } = deployment;
        const entityData = networkNodeMap[entity.id];
        // need to change edges to include external sources
        const edges = getEdgesFromNode(configObj);

        const { externallyConnected } = filterState === filterModes.all ? entityData.active : datum;

        const isSelected = !!(selectedNode?.id === entity.id);
        const isHovered = !!(hoveredNode?.id === entity.id);

        const isAdjacent = getIsAdjacentToHighlightedNode({
            selectedNode,
            hoveredNode,
            entityId: entity.id,
            filterState,
        });
        const isBackground =
            !isAdjacent && (selectedNode || hoveredNode) && !isHovered && !isSelected;

        const isNonIsolated = getIsNonIsolatedNode(datum);
        const isExternallyConnected = externallyConnected && filterState !== filterModes.allowed;
        const classes = getClasses({
            active: datum.isActive,
            selected: isSelected,
            deployment: true,
            hovered: isHovered,
            background: isBackground,
            nonIsolated: isNonIsolated,
            externallyConnected: isExternallyConnected,
        });

        const { ingress, egress } = getDirectionalityEdges(entityData, filterState);

        const deploymentNode = {
            data: {
                ...datumProps,
                ...entityProps,
                ...deploymentProps,
                parent: namespace,
                edges,
                deploymentId: entityProps.id,
                ingress,
                egress,
            },
            classes,
        };
        return deploymentNode;
    });

    return deploymentList;
};

/**
 * Iterates through the list of nodes to return the data of a single deployment
 *
 * @param {!string} id node id
 * @param {!Object[]} deploymentList list of deployments
 * @returns {!Object[]}
 */
export const getNodeData = (id, deploymentList) => {
    return deploymentList.filter((node) => node.data.deploymentId === id);
};

/**
 * Iterates through a list of links and returns all links for the currently interacted node
 *
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains links, filterState, and nodeSideMap,
 *                            hoveredNode, and selectedNode
 * @returns {!Object[]}
 */
export const getEdges = (configObj) => {
    const { shouldShowNamespaceEdges } = configObj;
    const namespaceEdges = shouldShowNamespaceEdges ? getNamespaceEdges(configObj) : [];
    const edgesFromNodes = getEdgesFromNode(configObj);
    return [...namespaceEdges, ...edgesFromNodes];
};

/**
 * Iterates through the nodes to return a list of namespaces with active deployments
 *
 * @param {!Object} filteredData nodes that pertain to deployments
 * @param {!Object[]} deploymentList list of deployments
 * @returns {!Object[]}
 */
export const getActiveNamespaceList = (filteredData, deploymentList) => {
    return uniq(
        filteredData.reduce((acc, curr) => {
            const nsName = curr.entity.deployment.namespace;
            if (
                deploymentList.some(
                    (element) => element.data.isActive && element.data.parent === nsName
                )
            ) {
                acc.push(nsName);
            }

            return acc;
        }, [])
    );
};

/**
 * Iterates through a list of nodes to return a list of namespaces enriched by styling classes
 *
 * @param {!Object} filteredData nodes that pertain to deployments
 * @param {!Object[]} deploymentList list of deployments
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains hoveredNode, and selectedNode
 * @returns {!Object[]}
 */
export const getNamespaceList = (
    filteredData,
    deploymentList,
    { hoveredNode, selectedNode },
    cluster,
    filterState
) => {
    const activeNamespaceList = getActiveNamespaceList(filteredData, deploymentList);
    const highlightedNamespaces = {};
    const adjacentNodeIdsToHoveredNode = getAdjacentNodeIdsToNode(hoveredNode, filterState);
    const adjacentNodeIdsToSelectedNode = getAdjacentNodeIdsToNode(selectedNode, filterState);
    const namespaceList = uniq(
        filteredData.map(({ entity }) => {
            const { namespace } = entity.deployment;
            if (!highlightedNamespaces[namespace]) {
                highlightedNamespaces[namespace] =
                    adjacentNodeIdsToHoveredNode?.find((id) => entity.id === id) ||
                    adjacentNodeIdsToSelectedNode?.find((id) => entity.id === id);
            }
            return namespace;
        })
    ).map((namespace) => {
        const isActive = activeNamespaceList.includes(namespace);
        const isHovered = hoveredNode?.id === namespace || hoveredNode?.parent === namespace;
        const isSelected = selectedNode?.id === namespace || selectedNode?.parent === namespace;
        const isAdjacent = highlightedNamespaces[namespace];
        const isBackground =
            !isAdjacent && (selectedNode || hoveredNode) && !isHovered && !isSelected;
        const classes = getClasses({
            nsGroup: true,
            nsActive: isActive,
            nsSelected: isSelected,
            nsHovered: isAdjacent || isHovered,
            background: isBackground,
        });

        return {
            data: {
                id: namespace,
                name: `${isActive ? '\ue901 ' : ''}${namespace}`,
                active: isActive,
                type: entityTypes.NAMESPACE,
                parent: cluster,
            },
            classes,
        };
    });
    return namespaceList;
};

const sides = ['top', 'left', 'right', 'bottom'];

/**
 * Returns a list of edge nodes that are hidden cardinal direction edges
 *
 * @param {!String} id
 * @param {!String} classes
 * @param {!String} type
 * @returns {!Object[]}
 */
const createEdgeNodes = (id, classes, type) => {
    const emptyArray = [];
    const edgeNodes = sides.reduce((acc, side) => {
        const node = {
            data: {
                id: `${id}_${side}`,
                parent: id,
                side,
                category: type,
            },
            classes,
        };
        return [...acc, node];
    }, emptyArray);
    return edgeNodes;
};

export const getEdgeNodes = (nodeList, classes) => {
    const totalEdgeNodes = nodeList.reduce((acc, node) => {
        const { id, type } = node.data;
        const edgeNodes = createEdgeNodes(id, classes, type);
        return [...acc, ...edgeNodes];
    }, []);
    return totalEdgeNodes;
};

/**
 * Returns a list of nodes that are hidden "namespace" cardinal direction edges
 *
 * @param {!Object[]} namespaces list of namespaces
 *
 * @returns {!Object[]}
 */
export const getNamespaceEdgeNodes = (namespaces) => {
    const namespaceEdgeNodes = getEdgeNodes(namespaces, 'nsEdge');
    return namespaceEdgeNodes;
};

/**
 * Returns a list of nodes that are hidden "external entities" cardinal direction edges
 *
 * @param {!Object} externalEntitiesNode
 *
 * @returns {!Object[]}
 */
export const getExternalEntitiesEdgeNodes = (externalEntitiesNode) => {
    const externalEntitiesEdgeNodes = getEdgeNodes([externalEntitiesNode], 'externalEntitiesEdge');
    return externalEntitiesEdgeNodes;
};

/**
 * Returns a list of nodes that are hidden "external source" cardinal direction edges
 *
 * @param {!Object} cidrBlockNodes
 *
 * @returns {!Object[]}
 */
export const getCIDRBlockEdgeNodes = (cidrBlockNodes) => {
    const cidrBlockEdgeNodes = getEdgeNodes(cidrBlockNodes, 'cidrBlockEdge');
    return cidrBlockEdgeNodes;
};

/**
 * Grabs either the ingress or egress ports and protocols from the network flows
 *
 * @param {!Object[]} networkFlows
 * @param {!String} traffic
 * @returns {!Object[]}
 */
function getPortsAndProtocolsByDirectionality(networkFlows, traffic) {
    if (!networkFlows) {
        return [];
    }
    return networkFlows.reduce((acc, networkFlow) => {
        return [
            ...acc,
            ...networkFlow.portsAndProtocols.filter((datum) => datum.traffic === traffic),
        ];
    }, []);
}

/**
 * Grabs either the ingress ports and protocols from the network flows
 *
 * @param {!Object[]} networkFlows
 * @returns {!Object[]}
 */
export function getIngressPortsAndProtocols(networkFlows) {
    return getPortsAndProtocolsByDirectionality(networkFlows, networkTraffic.INGRESS);
}

/**
 * Grabs either the egress ports and protocols from the network flows
 *
 * @param {!Object[]} networkFlows
 * @returns {!Object[]}
 */
export function getEgressPortsAndProtocols(networkFlows) {
    return getPortsAndProtocolsByDirectionality(networkFlows, networkTraffic.EGRESS);
}

/**
 * Determines if the node is hoverable (like deployment or external entities)
 *
 * @param {string} type the type of our graph node
 *
 * @return {boolean}
 */
export function getIsNodeHoverable(type) {
    return (
        type === entityTypes.DEPLOYMENT ||
        type === nodeTypes.EXTERNAL_ENTITIES ||
        type === nodeTypes.CIDR_BLOCK
    );
}

/**
 * Attempt to get a more descriptive error message based on the text of the server
 * response, otherwise return a standard error message.
 *
 * @param {string} serverError The error message sent from the server.
 * @param {string=} defaultMessage A default message to use as a fallback for non-specific error cases.
 *
 * @returns {string} The user facing error message to display.
 */
export function getErrorMessageFromServerResponse(
    serverError,
    defaultMessage = 'Please refresh the page. If this problem continues, please contact support.'
) {
    // Attempt to detect scale issues from the server error. This typically occurs when filters
    // are selected that create a graph too large for the backend to calculate efficiently.
    if (serverError.includes('exceeds maximum')) {
        return 'To reload the graph, try updating the filters at the top of the page.';
    }
    return defaultMessage;
}
