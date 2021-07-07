import {
    LabelSelectorsKey,
    LabelSelectorOperator,
    LabelSelectorRequirement,
    getIsKeyExistsOperator,
} from 'services/RolesService';

export type LabelSelectorsEditingState = {
    labelSelectorsKey: LabelSelectorsKey; // tab key corresponds to data key
    indexLabelSelector: number;
};

export type Activity = 'DISABLED' | 'ENABLED' | 'ACTIVE';

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

/*
 * Return whether a label selector is ACTIVE to create or update; otherwise:
 * ENABLED to update, because no other label selector is active
 * DISABLED from update, because some other label selector is active
 */
export function getLabelSelectorActivity(
    labelSelectorsKey: LabelSelectorsKey,
    indexLabelSelectorActive: number,
    labelSelectorsEditingState: LabelSelectorsEditingState | null
): Activity {
    if (labelSelectorsEditingState) {
        if (labelSelectorsKey === labelSelectorsEditingState.labelSelectorsKey) {
            return indexLabelSelectorActive === labelSelectorsEditingState.indexLabelSelector
                ? 'ACTIVE'
                : 'DISABLED'; // because editing another label selector on this tab
        }

        return 'DISABLED'; // because editing on the other tab
    }

    return 'ENABLED'; // because not editing on either tab
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
