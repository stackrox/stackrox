export function getDeploymentAndProcessIdFromProcessGroup(group) {
    if (group && group.groups && group.groups.length) {
        const firstSubGroup = group.groups[0];
        if (firstSubGroup.signals && firstSubGroup.signals.length) {
            const firstSignal = firstSubGroup.signals[0];
            const { clusterId, namespace } = firstSignal;
            return { clusterId, namespace };
        }
    }
    return {};
}

export function getDeploymentAndProcessIdFromGroupedProcesses(groups) {
    // Derive the clusterId and namespace from the processes. Since all the processes are for the same deployment
    // we can just use the first one.
    if (groups && groups.length) {
        const firstGroup = groups[0];
        return getDeploymentAndProcessIdFromProcessGroup(firstGroup);
    }
    return {};
}
