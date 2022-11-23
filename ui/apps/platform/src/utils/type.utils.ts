/**
 * This function is not callable. This can be used to ensure at compile time that a given code
 * block cannot be reached, such as in the `default` case of a switch/case block to ensure
 * all possible cases are covered.
 */
export function ensureExhaustive(_: never): never {
    return _;
}

/**
 * Type guard to check if an array is empty, and if so, narrows the array
 * to a tuple that guarantees at least one element.
 */
export function isNonEmptyArray<T>(arr: T[]): arr is [T, ...T[]] {
    return arr.length > 0;
}

/**
 * The flip side of TypeScript's `keyof` operator, returns a union of
 * all property value types of a given object.
 */
export type ValueOf<T extends Record<string | number | symbol, unknown>> = T[keyof T];
