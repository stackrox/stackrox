export function getPercentage(num, total) {
    return !total ? 0 : Math.round((num / total) * 100);
}

export default {
    getPercentage
};
