import type {
    LabelSelector,
    LabelSelectorsKey,
    LabelSelectorOperator,
    LabelSelectorRequirement,
    SimpleAccessScopeRules,
} from 'services/AccessScopesService';

/*
 * Validation for simple access scopes.
 */

export function getIsKeyExistsOperator(op: LabelSelectorOperator): boolean {
    return op === 'EXISTS' || op === 'NOT_EXISTS';
}

export function getIsKeyInSetOperator(op: LabelSelectorOperator): boolean {
    return op === 'IN' || op === 'NOT_IN';
}

/*
 * A valid "key in set" requirement has at least one value.
 */
export function getIsValidRequirement({ op, values }: LabelSelectorRequirement): boolean {
    return !getIsKeyInSetOperator(op) || values.length !== 0;
}

/*
 * A valid label selector has at least one requirement.
 */
export function getIsValidRequirements(requirements: LabelSelectorRequirement[]): boolean {
    return requirements.length !== 0 && requirements.every(getIsValidRequirement);
}

export function getIsValidLabelSelectors(labelSelectors: LabelSelector[]): boolean {
    return labelSelectors.every(({ requirements }) => getIsValidRequirements(requirements));
}

export function getIsValidRules({
    clusterLabelSelectors,
    namespaceLabelSelectors,
}: SimpleAccessScopeRules): boolean {
    return (
        getIsValidLabelSelectors(clusterLabelSelectors) &&
        getIsValidLabelSelectors(namespaceLabelSelectors)
    );
}

function getTemporarilyValidLabelSelectors(labelSelectors: LabelSelector[]): LabelSelector[] {
    const temporarilyValidLabelSelectors: LabelSelector[] = [];

    labelSelectors.forEach((labelSelector) => {
        const { requirements } = labelSelector;
        if (requirements.length === 0 || getIsValidRequirements(requirements)) {
            // Although a label selector which has no requirements is not valid to save,
            // do not filter it out from temporarily valid state while adding or editing.
            temporarilyValidLabelSelectors.push(labelSelector);
        } else {
            // However, do filter out any set requirements which have no values.
            temporarilyValidLabelSelectors.push({
                requirements: requirements.filter(getIsValidRequirement),
            });
        }
    });

    return temporarilyValidLabelSelectors;
}

/*
 * If rules are temporarily invalid while adding or editing label selectors,
 * return rules that are valid for computeeffectiveaccessscope request.
 */
export function getTemporarilyValidRules(rules: SimpleAccessScopeRules): SimpleAccessScopeRules {
    const { clusterLabelSelectors, namespaceLabelSelectors } = rules;

    return {
        ...rules,
        clusterLabelSelectors: getTemporarilyValidLabelSelectors(clusterLabelSelectors),
        namespaceLabelSelectors: getTemporarilyValidLabelSelectors(namespaceLabelSelectors),
    };
}

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

export function getOpText(op: LabelSelectorOperator, values: string[]): string {
    switch (op) {
        case 'IN':
            return values.length > 1 ? 'in' : '=';
        case 'NOT_IN':
            return values.length > 1 ? 'not in' : '!=';
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
