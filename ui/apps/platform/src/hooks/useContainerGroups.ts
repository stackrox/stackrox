import useFeatureFlags from 'hooks/useFeatureFlags';
import type { ContainerType } from 'types/deployment.proto';

export default function useContainerGroups<
    T extends {
        type: ContainerType;
    },
>(containers: T[] | null | undefined) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showInitContainers = isFeatureFlagEnabled('ROX_INIT_CONTAINER_SUPPORT');
    const all = containers ?? [];
    return {
        regularContainers: all.filter((c) => c.type !== 'INIT' || !showInitContainers),
        initContainers: all.filter((c) => c.type === 'INIT' && showInitContainers),
    };
}
