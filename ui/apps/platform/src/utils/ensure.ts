/**
 * Helper function to ensure that the provided value is a string.
 *
 * If the value is a string, it returns the value as is.
 * If the value is not a string, it returns an empty string.
 *
 * @param value - The value to check and ensure as a string
 * @returns {string} - The original value if it's a string, otherwise an empty string
 */
export function ensureString(value: unknown): string {
    if (typeof value === 'string') {
        return value;
    }
    return '';
}

/**
 * Helper function to ensure that the provided value is converted to an array of strings.
 *
 * Example:
 * ensureStringArray(['a', 'b', 'c']);   // returns ['a', 'b', 'c']
 * ensureStringArray('hello');           // returns ['hello']
 * ensureStringArray([1, 'b', {}]);      // returns ['b'] (filters out non-string elements)
 * ensureStringArray(123);               // returns []
 *
 * @param value - The input value, which could be of any type.
 * @returns {string[]} - If the value is an array, it filters and returns only the string elements.
 *                       If the value is a single string, it wraps it in an array and returns it.
 *                       For all other values, it returns an empty array.
 */
export function ensureStringArray(value: unknown): string[] {
    if (Array.isArray(value)) {
        const result: string[] = value.filter((element) => typeof element === 'string');
        return result;
    }
    if (typeof value === 'string') {
        return [value];
    }
    return [];
}

/**
 * Helper function to ensure that the provided value is converted to a boolean.
 *
 * Example:
 * ensureBoolean(true);        // returns true
 * ensureBoolean('true');      // returns true
 * ensureBoolean('false');     // returns false
 * ensureBoolean(123);         // returns false (non-boolean values default to false)
 *
 * @param value - The input value, which could be of any type.
 * @returns {boolean} - Returns the value as a boolean if it's already a boolean,
 *                      or converts string values 'true'/'false' (case-insensitive) to booleans.
 *                      Returns `false` for all other values.
 */
export function ensureBoolean(value: unknown): boolean {
    if (typeof value === 'boolean') {
        return value;
    }
    if (typeof value === 'string' && value.toLowerCase() === 'true') {
        return true;
    }
    if (typeof value === 'string' && value.toLowerCase() === 'false') {
        return false;
    }
    return false;
}
