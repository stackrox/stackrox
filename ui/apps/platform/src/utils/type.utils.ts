/**
 * This function is not callable. This can be used to ensure at compile time that a given code
 * block cannot be reached, such as in the `default` case of a switch/case block to ensure
 * all possible cases are covered.
 */
export function ensureExhaustive(_: never): never {
    return _;
}

export type NonEmptyArray<T> = [T, ...T[]];

/**
 * Type guard to check if an array is empty, and if so, narrows the array
 * to a tuple that guarantees at least one element.
 */
export function isNonEmptyArray<T>(arr: T[]): arr is NonEmptyArray<T> {
    return arr.length > 0;
}

/**
 * The flip side of TypeScript's `keyof` operator, returns a union of
 * all property value types of a given object.
 */
export type ValueOf<T extends Record<string | number | symbol, unknown>> = T[keyof T];

/**
 * Overrides properties with type intersection
 *
 */
export type Override<T1, T2> = Omit<T1, keyof T2> & T2;

/**
 * A predicate function used to determine if an item is `null` or `undefined`. If it is not
 * nullish, the function will safely narrow the type to the non-nullish type.
 *
 * @example
 * const mixedArray = ['test','test2', null]; // Type is (string | null)[]
 * const naiveFilteredArray = mixedArray.filter((item) => item !== null); // Type is still (string | null)[]
 * const filteredArray = mixedArray.filter(isNonNullish); // Type is string[]
 */
export function isNonNullish<T>(val: T | null | undefined): val is T {
    return val !== null && typeof val !== 'undefined';
}
