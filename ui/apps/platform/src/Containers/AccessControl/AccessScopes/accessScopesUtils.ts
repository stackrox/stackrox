import { LabelSelectorsKey } from 'services/RolesService';

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
