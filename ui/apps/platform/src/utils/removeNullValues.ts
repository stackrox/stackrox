// The type doesn't matter here. We want to remove null values
export function removeNullValues<T>(arr: T[]): T[] {
    return arr.filter((item) => item !== null);
}
