/* eslint-disable @typescript-eslint/no-unsafe-return */
import { NetworkNode, SimulatedBaseline, EntityType } from 'Containers/Network/networkTypes';
import { getIsExternal, getSourceTargetKey } from 'utils/networkGraphUtils';

// @TODO: This type might be something we can re-use
export interface NetworkLink {
    isActive: boolean;
    isAllowed: boolean;
    isBetweenNonIsolated: boolean;
    isExternal: boolean;
    source: string;
    sourceNS: string;
    sourceName: string;
    sourceType: EntityType;
    target: string;
    targetNS: string;
    targetName: string;
    targetType: EntityType;
}

interface NetworkLinkWithSimulatedStatus extends NetworkLink {
    isSimulated: boolean;
    simulatedStatus: 'REMOVED' | 'ADDED' | 'UNMODIFIED' | 'MODIFIED';
}

function getMergedSimulatedStatus(prevStatus, newStatus) {
    if (!prevStatus) {
        return newStatus;
    }
    if (
        (prevStatus === 'ADDED' && newStatus !== 'ADDED') ||
        (prevStatus === 'REMOVED' && newStatus !== 'REMOVED') ||
        (prevStatus === 'UNMODIFIED' && newStatus !== 'UNMODIFIED')
    ) {
        return 'MODIFIED';
    }
    return prevStatus;
}

function enhanceLinksWithSimulatedStatus(
    selectedNode: NetworkNode,
    links: NetworkLink[],
    simulatedBaselines: SimulatedBaseline[]
): NetworkLink[] | NetworkLinkWithSimulatedStatus[] {
    const selectedNodeId = selectedNode.id;

    const simulatedStatusMap = simulatedBaselines.reduce((acc, curr) => {
        const currNodeId = curr.peer.entity.id;
        const key = getSourceTargetKey(selectedNodeId, currNodeId);
        acc[key] = getMergedSimulatedStatus(acc[key], curr.simulatedStatus);
        return acc;
    }, {});

    const linksWithSimulatedBaselines = links.map((link) => {
        const sourceTargetKey = getSourceTargetKey(link.source, link.target);
        const simulatedStatus = simulatedStatusMap[sourceTargetKey];
        if (!simulatedStatus) {
            return link;
        }
        const modifiedLink = { ...link, isSimulated: true, simulatedStatus };
        return modifiedLink;
    });

    const addedSimulatedBaselines = simulatedBaselines.filter((baseline) => {
        const currNodeId = baseline.peer.entity.id;
        const key = getSourceTargetKey(selectedNodeId, currNodeId);
        return simulatedStatusMap[key] === 'ADDED';
    });
    const addedLinks = addedSimulatedBaselines.reduce(
        (acc, curr): NetworkLinkWithSimulatedStatus[] => {
            const {
                peer: { entity, ingress },
            } = curr;
            const linkData = ingress
                ? {
                      source: selectedNode.id,
                      sourceNS: selectedNode.parent,
                      sourceName: selectedNode.name,
                      sourceType: selectedNode.type,
                      target: entity.id,
                      targetNS: getIsExternal(entity.type) ? entity.id : entity.namespace,
                      targetName: entity.name,
                      targetType: entity.type,
                  }
                : {
                      source: entity.id,
                      sourceNS: getIsExternal(entity.type) ? entity.id : entity.namespace,
                      sourceName: entity.name,
                      sourceType: entity.type,
                      target: selectedNode.id,
                      targetNS: selectedNode.parent,
                      targetName: selectedNode.name,
                      targetType: selectedNode.type,
                  };
            const result = {
                isActive: false,
                isAllowed: true,
                isBetweenNonIsolated: false,
                isExternal: false,
                isSimulated: true,
                simulatedStatus: 'ADDED',
                ...linkData,
            } as NetworkLinkWithSimulatedStatus;
            return [...acc, result];
        },
        [] as NetworkLinkWithSimulatedStatus[]
    );

    const newLinks = [...linksWithSimulatedBaselines, ...addedLinks];

    return newLinks;
}

export default enhanceLinksWithSimulatedStatus;
