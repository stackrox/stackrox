import useFeatureFlags from './useFeatureFlags';

export default function useIsScannerV4Enabled() {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    return isFeatureFlagEnabled('ROX_SCANNER_V4');
}
