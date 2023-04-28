/* eslint-disable @typescript-eslint/no-unsafe-return */
// The type doesn't matter here. We want to remove null values
export function removeNullValues(arr) {
    return arr.filter((item) => item !== null);
}
