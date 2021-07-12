import {
    LabelSelectorsKey,
    LabelSelectorOperator,
    LabelSelectorRequirement,
    getIsKeyExistsOperator,
} from 'services/RolesService';

/*
 * Each tab key has value either index of a label selector while (adding or) editing;
 * otherwise -1 if neither adding nor editing.
 */
export type LabelSelectorsEditingState = {
    clusterLabelSelectors: number;
    namespaceLabelSelectors: number;
};

export function getIsEditingLabelSelectors({
    clusterLabelSelectors,
    namespaceLabelSelectors,
}: LabelSelectorsEditingState): boolean {
    return clusterLabelSelectors !== -1 || namespaceLabelSelectors !== -1;
}

export function getIsEditingLabelSelectorOnTab(
    labelSelectorsEditingState: LabelSelectorsEditingState,
    labelSelectorsKey: LabelSelectorsKey
): boolean {
    return labelSelectorsEditingState[labelSelectorsKey] !== -1;
}

export type Activity = 'DISABLED' | 'ENABLED' | 'ACTIVE';

/*
 * Return whether a label selector is ACTIVE to create or update; otherwise:
 * ENABLED to update, because no other label selector is active
 * DISABLED from update, because some other label selector is active
 */
export function getLabelSelectorActivity(
    labelSelectorsEditingState: LabelSelectorsEditingState,
    labelSelectorsKey: LabelSelectorsKey,
    indexLabelSelector: number
): Activity {
    if (labelSelectorsEditingState[labelSelectorsKey] === indexLabelSelector) {
        return 'ACTIVE';
    }

    return labelSelectorsEditingState[labelSelectorsKey] === -1
        ? 'ENABLED' // because not editing on this tab
        : 'DISABLED'; // because editing another label selector on this tab
}

/*
 * Return whether a requirement is ACTIVE to create or update; otherwise:
 * ENABLED to update, because no other requirement is active
 * DISABLED from update, because some other requirement is active
 */
export function getRequirementActivity(index: number, indexActive: number): Activity {
    if (index === indexActive) {
        return 'ACTIVE';
    }

    return indexActive === -1 ? 'ENABLED' : 'DISABLED';
}

export function getOpText(op: LabelSelectorOperator): string {
    switch (op) {
        case 'IN':
            return 'in';
        case 'NOT_IN':
            return 'not in';
        case 'EXISTS':
            return 'exists';
        case 'NOT_EXISTS':
            return 'not exists';
        default:
            return 'unknown';
    }
}

export function getValueText(value: string): string {
    return value || '""';
}

export function getIsOverlappingRequirement(
    key: string,
    op: LabelSelectorOperator,
    requirements: LabelSelectorRequirement[]
): boolean {
    return requirements.some((requirement) => {
        if (key === requirement.key) {
            if (op === requirement.op) {
                // Prevent same key and op because:
                // redundant for exists
                // confusing for set, because effective requirement is intersection of values
                // however, "key in (...)" and "key not in (...)" are possible
                return true;
            }

            if (getIsKeyExistsOperator(op) && getIsKeyExistsOperator(requirement.op)) {
                // Prevent "key exists" and "key not exists" because they contradict each other.
                return true;
            }
        }

        return false;
    });
}
