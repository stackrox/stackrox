function getHasDuplicateFields(values, field) {
    const fieldValues = values.entities.map(({ entity }) => entity[field]);
    return new Set(fieldValues).size !== fieldValues.length;
}

export function getHasDuplicateCIDRNames(values) {
    return getHasDuplicateFields(values, 'name');
}

export function getHasDuplicateCIDRAddresses(values) {
    return getHasDuplicateFields(values, 'cidr');
}
