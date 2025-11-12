import type { Policy } from 'types/policy.proto';

/**
 * Extracts the group path from a field name like:
 * "policySections[0].policyGroups[1].values[2]" -> "policySections[0].policyGroups[1]"
 */
function getGroupPathFromFieldName(fieldName: string): string | null {
    const match = fieldName.match(/^(policySections\[\d+\]\.policyGroups\[\d+\])\.values\[\d+\]$/);
    return match ? match[1] : null;
}

/**
 * Parses a group path to extract section and group indices.
 * Example: "policySections[0].policyGroups[1]" -> { sectionIndex: 0, groupIndex: 1 }
 */
function parseGroupPath(groupPath: string): { sectionIndex: number; groupIndex: number } | null {
    const match = groupPath.match(/policySections\[(\d+)\]\.policyGroups\[(\d+)\]/);
    if (!match) {
        return null;
    }

    return {
        sectionIndex: parseInt(match[1], 10),
        groupIndex: parseInt(match[2], 10),
    };
}

/**
 * Extracts the value index from a field name.
 * Example: "policySections[0].policyGroups[1].values[2]" -> 2
 */
function extractValueIndex(fieldName: string): number {
    const match = fieldName.match(/\.values\[(\d+)\]/);
    return match ? parseInt(match[1], 10) : -1;
}

/**
 * Filters out already-selected options for direct field criteria.
 * Use this for criteria where the value is stored directly in `val.value`.
 * Example: "Port Exposure Method", "Add Capabilities", "Drop Capabilities"
 *
 * @param options - The full list of available options
 * @param fieldName - The field name (e.g., "policySections[0].policyGroups[1].values[2]")
 * @param values - The Formik policy values
 * @returns Filtered options excluding already-selected values
 */
export function getAvailableOptionsForField<T extends { value: string }>(
    options: T[],
    fieldName: string,
    values: Policy
): T[] {
    const alreadySelected = new Set<string>();
    const groupPath = getGroupPathFromFieldName(fieldName);
    if (!groupPath) {
        return options;
    }

    const indices = parseGroupPath(groupPath);
    if (!indices) {
        return options;
    }

    const { sectionIndex, groupIndex } = indices;
    const group = values.policySections[sectionIndex]?.policyGroups[groupIndex];
    if (!group) {
        return options;
    }

    const valueIndex = extractValueIndex(fieldName);
    group.values.forEach((val, idx) => {
        if (idx !== valueIndex && val.value && typeof val.value === 'string') {
            alreadySelected.add(val.value);
        }
    });

    return options.filter((option) => !alreadySelected.has(option.value));
}
