import type { ContainerType } from 'types/deployment.proto';

export default function useContainerGroups<T extends { type: ContainerType }>(
    containers: T[] | null | undefined
) {
    const all = containers ?? [];
    return {
        regularContainers: all.filter((c) => c.type !== 'INIT'),
        initContainers: all.filter((c) => c.type === 'INIT'),
    };
}
