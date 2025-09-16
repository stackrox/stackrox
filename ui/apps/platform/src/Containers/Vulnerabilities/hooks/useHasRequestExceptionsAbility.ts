import usePermissions from 'hooks/usePermissions';

export default function useHasRequestExceptionsAbility(): boolean {
    const { hasReadWriteAccess } = usePermissions();

    return hasReadWriteAccess('VulnerabilityManagementRequests');
}
