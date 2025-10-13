import { useState } from 'react';

export type SetResult<T> = {
    has: (key: T) => boolean;
    toggle: (key: T, isOn?: boolean) => void;
    clear: () => void;
    size: number;
    asArray: () => void;
};

/**
 * Hook that wraps a native `Set` for easier immutable usage as React component state
 *
 * Note that the API is intentionally limited - we should add more `Set` methods as use
 * cases require them.
 */

export default function useSet<T>(initialSet: Set<T> = new Set()): SetResult<T> {
    const [itemSet, setItemSet] = useState(initialSet);

    function has(key: T): boolean {
        return itemSet.has(key);
    }

    /**
     * Adds or removes an item from the set
     *
     * @param key
     *      The item to toggle
     * @param isOn
     *      Force the item to exist or not exist in the set. If this param is
     *      omitted the item will be toggled to the opposite of its current state
     */
    function toggle(key: T, isOn?: boolean) {
        setItemSet((prevSet) => {
            const nextSet = new Set(prevSet);
            const shouldAdd = typeof isOn === 'undefined' ? !itemSet.has(key) : isOn;
            if (shouldAdd) {
                nextSet.add(key);
            } else {
                nextSet.delete(key);
            }
            return nextSet;
        });
    }

    /**
     * Empties the set
     */
    function clear() {
        setItemSet(new Set());
    }

    function asArray() {
        return Array.from(itemSet);
    }

    return { has, toggle, clear, size: itemSet.size, asArray };
}
