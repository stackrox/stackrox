function objectToWhereClause(query) {
    if (!query) return '';

    return Object.entries(query)
        .reduce((acc, entry) => {
            const [key, value] = entry;
            if (!value) return acc;
            const flatValue = Array.isArray(value) ? value.join() : value;
            return `${acc}${key}:${flatValue}+`;
        }, '')
        .slice(0, -1);
}

export default {
    objectToWhereClause
};
