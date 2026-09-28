import useFeatureFlags from './useFeatureFlags';

export default function useIsLegacyScannerEnabled() {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    return isFeatureFlagEnabled('ROX_LEGACY_SCANNER');
}
