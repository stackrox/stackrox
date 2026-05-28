function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/**
 * Normalizes the storage keys to the canonical form to clean up mixed case, legacy keys.
 */
export function normalizeLocalStorageKeys(value: unknown): unknown {
    // Guard checks to ensure the value conforms to an expected JSON object structure.
    if (!isRecord(value)) {
        return value;
    }
    const { preferences } = value;
    if (!isRecord(preferences)) {
        return value;
    }

    const { defaultFilters } = preferences;
    if (!isRecord(defaultFilters)) {
        return value;
    }

    // Normalize the storage keys to the canonical form.
    let normalized = defaultFilters;
    if ('SEVERITY' in normalized) {
        const { SEVERITY, ...rest } = normalized;
        normalized = { ...rest, Severity: normalized.Severity ?? SEVERITY };
    }
    if ('FIXABLE' in normalized) {
        const { FIXABLE, ...rest } = normalized;
        normalized = { ...rest, Fixable: normalized.Fixable ?? FIXABLE };
    }

    if (normalized === defaultFilters) {
        return value;
    }
    return {
        ...value,
        preferences: { ...preferences, defaultFilters: normalized },
    };
}
