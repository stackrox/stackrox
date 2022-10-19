import { useState, useCallback } from 'react';

// Using a single value at the module scope allows multiple components to use this hook
// without generating conflicting keys.
let globalPrefix = 0;

type UseIndexKeyReturn = {
    /**
     * Generate a stable key for an array index
     */
    keyFor: (index: number) => string;
    /**
     * Invalidates all keys in use by this hook, causing React to rerender the entirety
     * of the list, but keeping render results predictable and correct.
     */
    invalidateIndexKeys: () => void;
};

/**
 * Hook that allows the usage of `index` values for React component keys in -specific- situations.
 * This should only be used when the following are true:
 * 1. There is no stable key to be used instead.
 * 2. The ordering of the items in the list never changes.
 *
 * For the second case, this hook can be used but the `invalidateKeys` function must be called whenever
 * the list is rearranged, or an item is deleted. This will eschew performance in favor of predictable
 * rendering.
 */
export default function useIndexKey(): UseIndexKeyReturn {
    const [prefix, setPrefix] = useState(() => {
        // Increment on initial call to hook, to ensure no two components start from the same index
        globalPrefix += 1;
        return globalPrefix;
    });
    const keyFor = useCallback((index: number) => `${prefix}-useIndexKey-${index}`, [prefix]);
    const invalidateIndexKeys = useCallback(() => {
        globalPrefix += 1;
        setPrefix(globalPrefix);
    }, [setPrefix]);

    return { keyFor, invalidateIndexKeys };
}
