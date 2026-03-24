export function getPercentage(num: number, total: number) {
    return !total ? 0 : Math.round((num / total) * 100);
}
