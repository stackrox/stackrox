// TODO: remove this helper function after VM Updates have been turned on
import entityTypes from 'constants/entityTypes';

export default function filterEntityRelationship(
    showVMUpdates: boolean,
    entityType: string
): boolean {
    if (showVMUpdates) {
        if (entityType === entityTypes.COMPONENT || entityType === entityTypes.CVE) {
            return false;
        }
    } else if (
        entityType === entityTypes.NODE_COMPONENT ||
        entityType === entityTypes.IMAGE_COMPONENT ||
        entityType === entityTypes.IMAGE_CVE ||
        entityType === entityTypes.NODE_CVE ||
        entityType === entityTypes.CLUSTER_CVE
    ) {
        return false;
    }
    return true;
}
