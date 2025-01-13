import usePermissions from 'hooks/usePermissions';

export default function useHasGenerateSBOMAbility() {
    const { hasReadWriteAccess } = usePermissions();

    return (
        // SBOM Generation mutates image scan state, so requires write access to 'Image'
        hasReadWriteAccess('Image')
    );
}
