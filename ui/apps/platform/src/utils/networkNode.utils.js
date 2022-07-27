import { nodeTypes } from 'constants/networkGraph';
import { filterModes } from 'constants/networkFilterModes';
import {
    getEdgesFromNode,
    getClasses,
    getIsAdjacentToHighlightedNode,
    getDirectionalityEdges,
} from './networkGraphUtils';

/**
 * Select out the entity representing external connections in the cluster
 *
 * @param   {!Object[]} data    list of "deployments", without the external entity filtered out
 * @param   {!Object} configObj config object of the current network graph state
 *                              that contains links, filterState, and nodeSideMap,
 *                              networkNodeMap, hoveredNode, and selectedNode
 *
 * @return  {!Object}
 */
export const getExternalEntitiesNode = (data, configObj = {}) => {
    const { hoveredNode, selectedNode, filterState, networkNodeMap } = configObj;

    const externalNode = data.find((datum) => datum?.entity?.type === nodeTypes.EXTERNAL_ENTITIES);

    if (!externalNode) {
        return null;
    }

    const { entity, ...datumProps } = externalNode;
    const entityData = networkNodeMap[entity.id];
    const edges = getEdgesFromNode(configObj);

    const externallyConnected =
        // TODO: figure out how this should be handled in External Entity context
        filterState === filterModes.all
            ? entityData?.active?.externallyConnected
            : externalNode?.externallyConnected;

    const isSelected = !!(selectedNode?.type === nodeTypes.EXTERNAL_ENTITIES);
    const isHovered = !!(hoveredNode?.type === nodeTypes.EXTERNAL_ENTITIES);
    const isAdjacent = getIsAdjacentToHighlightedNode({
        selectedNode,
        hoveredNode,
        entityId: entity.id,
        filterState,
    });
    const isBackground =
        !isAdjacent && !(!selectedNode && !hoveredNode) && !isHovered && !isSelected;
    // DEPRECATED: const isNonIsolated = getIsNonIsolatedNode(externalNode);
    // Historical note: isDisallowed was added in rox#2070 and then disabled in rox#2747
    /*
    const isDisallowed =
        filterState !== filterModes.allowed && edges.some((edge) => edge.data.isDisallowed);
    */
    const isExternallyConnected = externallyConnected && filterState !== filterModes.allowed;
    const classes = getClasses({
        active: false, // externalNode.isActive,
        nsSelected: isSelected,
        internet: true,
        nsHovered: isHovered,
        background: isBackground,
        nonIsolated: false,
        externallyConnected: isExternallyConnected,
    });

    const { ingress, egress } = getDirectionalityEdges(entityData, filterState);

    return {
        data: {
            ...datumProps,
            ...entity,
            id: entity.id,
            name: 'External Entities',
            active: false,
            edges,
            ingress,
            egress,
            type: nodeTypes.EXTERNAL_ENTITIES,
            parent: null,
        },
        classes,
    };
};

/**
 * Select out the entities representing external connections to CIDR blocks in the cluster
 *
 * @param   {!Object[]} data    list of "deployments", without the external entity filtered out
 * @param   {!Object} configObj config object of the current network graph state
 *                              that contains links, filterState, and nodeSideMap,
 *                              networkNodeMap, hoveredNode, and selectedNode
 *
 * @return  {!Object}
 */
export const getCIDRBlockNodes = (data, configObj = {}) => {
    const { hoveredNode, selectedNode, filterState, networkNodeMap } = configObj;

    const cidrBlocks = data.filter((datum) => datum?.entity?.type === nodeTypes.CIDR_BLOCK);

    if (cidrBlocks.length === 0) {
        return null;
    }

    const cidrBlockNodes = cidrBlocks.map((cidrBlock) => {
        const { entity, ...datumProps } = cidrBlock;
        const entityData = networkNodeMap[entity.id];
        const edges = getEdgesFromNode(configObj);

        const externallyConnected =
            filterState === filterModes.all
                ? entityData?.active?.externallyConnected
                : cidrBlock?.externallyConnected;

        const isSelected = !!(selectedNode?.id === entity.id);
        const isHovered = !!(hoveredNode?.id === entity.id);
        const isAdjacent = getIsAdjacentToHighlightedNode({
            selectedNode,
            hoveredNode,
            entityId: entity.id,
            filterState,
        });
        const isBackground =
            !isAdjacent && !(!selectedNode && !hoveredNode) && !isHovered && !isSelected;
        // DEPRECATED: const isNonIsolated = getIsNonIsolatedNode(externalNode);
        // Historical note: isDisallowed was added in rox#2070 and then disabled in rox#2747
        /*
        const isDisallowed =
            filterState !== filterModes.allowed && edges.some((edge) => edge.data.isDisallowed);
        */
        const isExternallyConnected = externallyConnected && filterState !== filterModes.allowed;
        const classes = getClasses({
            active: false,
            nsSelected: isSelected,
            cidrBlock: true,
            nsHovered: isHovered,
            background: isBackground,
            nonIsolated: false,
            externallyConnected: isExternallyConnected,
        });

        const { ingress, egress } = getDirectionalityEdges(entityData, filterState);

        return {
            data: {
                ...datumProps,
                ...entity,
                id: entity.id,
                cidr: entity.externalSource.cidr,
                name: entity.externalSource.name,
                edges,
                ingress,
                egress,
                active: false,
                type: nodeTypes.CIDR_BLOCK,
                parent: null,
            },
            classes,
        };
    });
    return cidrBlockNodes;
};

/**
 * Iterates through the networkNodeMap and returns the relevant list of nodes;
 *
 * @param {!Object} networkNodeMap map of nodes by nodeId
 * @param {number} filterState current filter state of the network graph
 * @returns {!Object[]}
 */
export const getFilteredNodes = (networkNodeMap, filterState) => {
    const nodes = [];
    Object.keys(networkNodeMap).forEach((nodeId) => {
        const { active: activeNode, allowed: allowedNode } = networkNodeMap[nodeId];
        if (filterState === filterModes.allowed) {
            if (allowedNode) {
                nodes.push(allowedNode);
            }
            return;
        }
        if (filterState === filterModes.active) {
            if (activeNode) {
                activeNode.policyIds = allowedNode?.policyIds?.flat() || [];
                nodes.push(activeNode);
            }
            return;
        }

        // "all" mode
        // We always expect an active node for the given entity,
        // but the allowed node may not be there for certain external entities.
        // We want to keep outEdges from both, but rely on the allowedNode for
        // properties like policyIds, nonIsolatedIngress, nonIsolatedEgress.
        if (!activeNode) {
            return;
        }
        // No allowed node, so just use the active node directly.
        if (!allowedNode) {
            nodes.push(activeNode);
            return;
        }
        const compositeNode = { ...allowedNode };
        // Merge outEdges from active into allowed.
        // like policyIds, nonIsolatedEgress etc from the allowed.
        Object.keys(activeNode.outEdges).forEach((targetEntityId) => {
            if (compositeNode.outEdges[targetEntityId]?.properties) {
                compositeNode.outEdges[targetEntityId].properties.push(
                    ...activeNode.outEdges[targetEntityId].properties
                );
            } else {
                compositeNode.outEdges[targetEntityId] = {
                    properties: activeNode.outEdges[targetEntityId].properties,
                };
            }
        });
        nodes.push(compositeNode);
    });
    return nodes;
};
