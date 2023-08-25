export function checkArrayContainsArray(allowedArray: string[], candidateArray: string[]) {
    return candidateArray.every((candidate) => allowedArray.includes(candidate));
}
