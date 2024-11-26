import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';

export default function useHasGenerateSBOMAbility() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { hasReadWriteAccess } = usePermissions();

    return (
        // Gate functionality for incremental implementation
        isFeatureFlagEnabled('ROX_SBOM_GENERATION') &&
        // SBOM Generation mutates image scan state, so requires write access to 'Image'
        hasReadWriteAccess('Image')
    );
}
