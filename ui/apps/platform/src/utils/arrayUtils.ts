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
