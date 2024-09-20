/**
 * Helper function to ensure that the provided value is a string.
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
