import entityTypes from 'constants/entityTypes';
import { nodeTypes } from 'constants/networkGraph';
import { UIfeatureFlags, isBackendFeatureFlagEnabled, knownBackendFlags } from 'utils/featureFlags';
import {
    getIsNodeHoverable,
    getSourceTargetKey,
    getNodeNamespace,
    getNodeName,
} from 'utils/networkGraphUtils';

/**
 * Iterates through a list of nodes and returns only links in the same namespace
 *
 * @param {!Object[]} nodes list of nodes
 * @param {!Object} networkEdgeMap map of edges in the graph by srcId--tgtId key
 * @param {!Object} networkNodeMap map of nodes in the graph by nodeId
 * @param {!string} filterState current filter state
 * @param {!string[]} featureFlags featureFlags
 * @returns {!Object[]}
 */
export const getLinks = (nodes, networkEdgeMap, networkNodeMap, filterState, featureFlags) => {
    const filteredLinks = [];
    // a map of all the edges in the node set to know whether we need to add disallowed edges
    const filteredEdgeHashTable = {};

    const isActive = (edgeKey) => !!networkEdgeMap[edgeKey]?.active;
    const isNonIsolated = (nodeId) => !!networkNodeMap[nodeId]?.nonIsolated;
    const isBetweenNonIsolated = (sourceId, targetId) =>
        isNonIsolated(sourceId) && isNonIsolated(targetId);
    const isAllowed = (edgeKey, { source, target, targetNS, sourceNS }) =>
        sourceNS === 'stackrox' ||
        targetNS === 'stackrox' ||
        isBetweenNonIsolated(source, target) ||
        !!networkEdgeMap[edgeKey]?.allowed;
    const isDisallowed = (edgeKey, link) =>
        UIfeatureFlags.SHOW_DISALLOWED_CONNECTIONS &&
        isActive(edgeKey) &&
        !isAllowed(edgeKey, link);

    const showExternalSources = isBackendFeatureFlagEnabled(
        featureFlags,
        knownBackendFlags.ROX_NETWORK_GRAPH_EXTERNAL_SRCS,
        false
    );

    nodes.forEach((node) => {
        const isHoverable = getIsNodeHoverable(node?.entity?.type);
        if (!isHoverable || !networkEdgeMap) {
            return;
        }

        const { id: sourceEntityId, type: sourceNodeType } = node.entity;
        const sourceNS = getNodeNamespace(node);
        const sourceName = getNodeName(node);
        const isSourceExternal =
            sourceNodeType === nodeTypes.EXTERNAL_ENTITIES ||
            sourceNodeType === nodeTypes.CIDR_BLOCK;

        // For nodes that are egress non-isolated, add outgoing edges to ingress non-isolated nodes, as long as the pair
        // of nodes is not fully non-isolated. This is a compromise to make the non-isolation highlight only apply in
        // the case when there are neither ingress nor egress policies (the data sent from the backend is optimized to
        // treat both phenomena separately and omit edges from a egress non-isolated to an ingress non-isolated
        // deployment, but that would be too confusing in the UI).
        if (node.nonIsolatedEgress) {
            nodes.forEach((targetNode) => {
                if (Object.is(node, targetNode)) {
                    return;
                }

                const { id: targetEntityId, type: targetNodeType } = targetNode.entity;
                const targetNS = getNodeNamespace(targetNode);
                const targetName = getNodeName(targetNode);
                const edgeKey = getSourceTargetKey(sourceEntityId, targetEntityId);

                if (
                    targetNode?.entity?.type !== entityTypes.DEPLOYMENT ||
                    !targetNode.nonIsolatedIngress // nodes that are ingress-isolated have explicit incoming edges
                ) {
                    return;
                }

                const isTargetExternal =
                    targetNodeType === nodeTypes.EXTERNAL_ENTITIES ||
                    targetNodeType === nodeTypes.CIDR_BLOCK;

                const link = {
                    source: sourceEntityId,
                    target: targetEntityId,
                    sourceName,
                    targetName,
                    sourceNS,
                    targetNS,
                    sourceType: sourceNodeType,
                    targetType: targetNodeType,
                    isExternal: isSourceExternal || isTargetExternal,
                };

                link.isActive = isActive(edgeKey);
                link.isBetweenNonIsolated = isBetweenNonIsolated(sourceEntityId, targetEntityId);
                link.isAllowed = isAllowed(edgeKey, link);
                link.isDisallowed = isDisallowed(edgeKey, link);

                // Do not draw implicit links between fully non-isolated nodes unless the connection is active.
                const isImplicit = node.nonIsolatedIngress && targetNode.nonIsolatedEgress;
                if (!isImplicit || link.isActive) {
                    filteredLinks.push(link);
                    filteredEdgeHashTable[edgeKey] = true;
                }
            });
        }

        Object.keys(node.outEdges).forEach((targetNodeId) => {
            const targetNode =
                networkNodeMap[targetNodeId].active || networkNodeMap[targetNodeId].allowed;
            const { id: targetEntityId, type: targetNodeType } = targetNode.entity;
            if (targetNodeType !== entityTypes.DEPLOYMENT && !showExternalSources) {
                return;
            }
            const edgeKey = getSourceTargetKey(sourceEntityId, targetEntityId);
            const targetNS = getNodeNamespace(targetNode);
            const targetName = getNodeName(targetNode);
            const isTargetExternal =
                targetNodeType === nodeTypes.EXTERNAL_ENTITIES ||
                targetNodeType === nodeTypes.CIDR_BLOCK;

            const link = {
                source: sourceEntityId,
                target: targetEntityId,
                sourceName,
                targetName,
                sourceNS,
                targetNS,
                sourceType: sourceNodeType,
                targetType: targetNodeType,
                isExternal: isSourceExternal || isTargetExternal,
            };

            link.isActive = isActive(edgeKey);
            link.isBetweenNonIsolated = isBetweenNonIsolated(sourceEntityId, targetEntityId);
            link.isAllowed = isAllowed(edgeKey, link);
            link.isDisallowed = isDisallowed(edgeKey, link);

            filteredLinks.push(link);
            filteredEdgeHashTable[edgeKey] = true;
        });
    });

    return filteredLinks;
};

/**
 * get links that are NOT external entities
 *
 * @param   {!Object[]}  links  edges betweem nodes
 *
 * @return  {!Object[]}
 */
export const getFilteredLinks = (links) => {
    const filteredLinks = [];
    links.forEach((link) => {
        if (link.sourceType !== 'INTERNET' && link.targetType !== 'INTERNET') {
            filteredLinks.push(link);
        }
    });
    return filteredLinks;
};
