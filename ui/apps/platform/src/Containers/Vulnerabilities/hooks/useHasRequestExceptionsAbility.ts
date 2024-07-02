import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';

export default function useHasRequestExceptionsAbility(): boolean {
    const { hasReadWriteAccess } = usePermissions();
    const { isFeatureFlagEnabled } = useFeatureFlags();

    return (
        hasReadWriteAccess('VulnerabilityManagementRequests') &&
        isFeatureFlagEnabled('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL')
    );
}
