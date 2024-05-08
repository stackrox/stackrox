import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';

export default function useCanSnoozeCves(): boolean {
    const { hasReadWriteAccess } = usePermissions();
    const { isFeatureFlagEnabled } = useFeatureFlags();

    return (
        hasReadWriteAccess('VulnerabilityManagementApprovals') &&
        isFeatureFlagEnabled('ROX_VULN_MGMT_LEGACY_SNOOZE')
    );
}
