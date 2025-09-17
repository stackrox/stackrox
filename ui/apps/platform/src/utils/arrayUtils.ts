export function checkArrayContainsArray(allowedArray: string[], candidateArray: string[]) {
    return candidateArray.every((candidate) => allowedArray.includes(candidate));
}

export function toggleItemInArray<T>(
    array: T[],
    selectedItem: T,
    isEqual: (a: T, b: T) => boolean = (a, b) => a === b
): T[] {
    const isIncluded = array.some((item) => isEqual(item, selectedItem));
    const newArray = isIncluded
        ? array.filter((item) => !isEqual(item, selectedItem))
        : [...array, selectedItem];
    return newArray;
}

/**
 * Normalizes a value to an array. If the value is already an array, returns it as-is.
 * If it's a single value, wraps it in an array.
 * @param value - The value to normalize to an array
 * @returns An array containing the value(s)
 */
export function normalizeToArray<T>(value: T | T[]): T[] {
    return Array.isArray(value) ? value : [value];
}
