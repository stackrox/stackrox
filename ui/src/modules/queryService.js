function objectToWhereClause(query) {
    if (!query) return '';

    return Object.entries(query)
        .reduce((acc, entry) => {
            const [key, value] = entry;
            if (typeof value === 'undefined' || value === '') return acc;
            const flatValue = Array.isArray(value) ? value.join() : value;
            const needsExactMatch = key.toLowerCase().indexOf(' id') !== -1;
            const queryValue = needsExactMatch ? `"${flatValue}"` : flatValue;
            return `${acc}${key}:${queryValue}+`;
        }, '')
        .slice(0, -1);
}

function entityContextToQueryObject(entityContext) {
    if (!entityContext) return {};

    return Object.keys(entityContext).reduce((acc, key) => {
        return { ...acc, [`${key} ID`]: entityContext[key] };
    }, {});
}

function getEntityWhereClause(search, entityContext) {
    return objectToWhereClause({ ...search, ...entityContextToQueryObject(entityContext) });
}

export default {
    objectToWhereClause,
    entityContextToQueryObject,
    getEntityWhereClause
};
