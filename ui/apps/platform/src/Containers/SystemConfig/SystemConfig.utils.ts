/**
 * Converts between bytes (B) and megabytes (MB).
 *
 * @param value - The value to convert.
 * @param fromUnit - The unit to convert from. Accepts 'B' for bytes or 'MB' for megabytes.

 * @returns The converted value.
 */
export function convertBetweenBytesAndMB(value: number, fromUnit: 'B' | 'MB'): number {
    const conversionFactor = 2 ** 20;

    return fromUnit === 'B' ? value / conversionFactor : value * conversionFactor;
}
