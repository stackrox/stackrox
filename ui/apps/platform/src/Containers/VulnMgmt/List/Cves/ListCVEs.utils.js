import uniq from 'lodash/uniq';

import entityTypes from 'constants/entityTypes';

// eslint-disable-next-line no-unused-vars
export function getFilteredCVEColumns(columns, workflowState, _isFeatureFlagEnabled) {
    const shouldKeepFixedByColumn =
        workflowState.isPreceding(entityTypes.IMAGE_COMPONENT) ||
        workflowState.isCurrentSingle(entityTypes.IMAGE_COMPONENT) ||
        workflowState.isPreceding(entityTypes.NODE_COMPONENT) ||
        workflowState.isCurrentSingle(entityTypes.NODE_COMPONENT);

    const shouldKeepDiscoveredAtImageColumn =
        workflowState.isPreceding(entityTypes.IMAGE) ||
        workflowState.isCurrentSingle(entityTypes.IMAGE) ||
        workflowState.getSingleAncestorOfType(entityTypes.IMAGE);

    const currentEntityType = workflowState.getCurrentEntityType();

    const shouldKeepDiscoveredTime = true;

    // No need to show entities in the node component or cluster context.
    const shouldKeepEntitiesColumn =
        !workflowState.isPrecedingSingle(entityTypes.NODE_COMPONENT) ||
        !workflowState.getSingleAncestorOfType(entityTypes.NODE);

    const shouldKeepSeverity =
        currentEntityType === entityTypes.IMAGE_CVE || currentEntityType === entityTypes.NODE_CVE;

    return columns.filter((col) => {
        switch (col.accessor) {
            case 'vulnerabilityTypes': {
                return false;
            }
            case 'fixedByVersion': {
                return shouldKeepFixedByColumn;
            }
            case 'createdAt': {
                return shouldKeepDiscoveredTime;
            }
            case 'discoveredAtImage': {
                return shouldKeepDiscoveredAtImageColumn;
            }
            case 'entities': {
                return shouldKeepEntitiesColumn;
            }
            case 'severity': {
                return shouldKeepSeverity || shouldKeepDiscoveredAtImageColumn;
            }
            default: {
                return true;
            }
        }
    });
}

export function parseCveNamesFromIds(cveIds) {
    const cveNames = cveIds.map((cveId) => {
        return cveId.split('#')[0];
    });

    return uniq(cveNames);
}

export default {
    getFilteredCVEColumns,
    parseCveNamesFromIds,
};
