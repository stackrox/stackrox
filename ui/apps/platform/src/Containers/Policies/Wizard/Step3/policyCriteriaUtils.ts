import type { ClientPolicy } from 'types/policy.proto';

/**
 * Parses a field name to extract section, group, and value indices.
 * Example: "policySections[0].policyGroups[1].values[2]" -> { sectionIndex: 0, groupIndex: 1, valueIndex: 2 }
 *
 * @returns Object with indices, or null if the field name doesn't match the expected pattern
 */
function parsePolicySectionFieldIndices(
    fieldName: string
): { sectionIndex: number; groupIndex: number; valueIndex: number } | null {
    const match = fieldName.match(
        /^policySections\[(\d+)\]\.policyGroups\[(\d+)\]\.values\[(\d+)\]$/
    );
    if (!match) {
        return null;
    }

    return {
        sectionIndex: parseInt(match[1], 10),
        groupIndex: parseInt(match[2], 10),
        valueIndex: parseInt(match[3], 10),
    };
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
    values: Pick<ClientPolicy, 'policySections'>
): T[] {
    const indices = parsePolicySectionFieldIndices(fieldName);
    if (!indices) {
        return options;
    }

    const { sectionIndex, groupIndex, valueIndex } = indices;
    const group = values.policySections[sectionIndex]?.policyGroups[groupIndex];
    if (!group) {
        return options;
    }

    const alreadySelected = new Set<string>();
    group.values.forEach((val, idx) => {
        if (idx !== valueIndex && val.value && typeof val.value === 'string') {
            alreadySelected.add(val.value);
        }
    });

    return options.filter((option) => !alreadySelected.has(option.value));
}
