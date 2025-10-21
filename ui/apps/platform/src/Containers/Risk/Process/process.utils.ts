import type { ProcessNameAndContainerNameGroup } from 'services/ProcessService';

export type ClusterIdAndNamespace = {
    clusterId: string;
    namespace: string;
};

export function getClusterIdAndNamespaceFromProcessGroup(
    group: ProcessNameAndContainerNameGroup | undefined
): ClusterIdAndNamespace {
    if (group && group.groups && group.groups.length) {
        const firstSubGroup = group.groups[0];
        if (firstSubGroup.signals && firstSubGroup.signals.length) {
            const firstSignal = firstSubGroup.signals[0];
            const { clusterId, namespace } = firstSignal;
            return { clusterId, namespace };
        }
    }
    return {
        clusterId: '',
        namespace: '',
    };
}

export function getClusterIdAndNamespaceFromGroupedProcesses(
    groups: ProcessNameAndContainerNameGroup[] | undefined
): ClusterIdAndNamespace {
    // Derive the clusterId and namespace from the processes. Since all the processes are for the same deployment
    // we can just use the first one.
    if (groups && groups.length) {
        const firstGroup = groups[0];
        return getClusterIdAndNamespaceFromProcessGroup(firstGroup);
    }
    return {
        clusterId: '',
        namespace: '',
    };
}
