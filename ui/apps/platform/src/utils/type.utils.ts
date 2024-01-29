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

export type JsonPrimitive = string | number | boolean | null;
export type JsonObject = { [key in string]: JsonValue } & {
    [Key in string]?: JsonValue | undefined;
};
export type JsonArray = JsonValue[];
/**
 * A type that represents any value that can be serialized to JSON.
 */
export type JsonValue = JsonPrimitive | JsonObject | JsonArray;

/**
 * Creates a type guard that checks if a value is one of a provided list of strings.
 *
 * @param values A const array of strings
 * @returns A type guard that checks if a value is one of the provided strings
 */
export function tupleTypeGuard<const T extends readonly string[]>(
    values: T
): (value: unknown) => value is T[number] {
    return (value: unknown): value is T[number] => values.some((arg) => arg === value);
}

export type UnionFrom<T extends readonly string[]> = T[number];
