import uniq from 'lodash/uniq';

function getAggregateText(leafValues: string[], multiplePhrase = 'Many'): string {
    const uniqValues = uniq(leafValues);
    if (uniqValues.length > 1) {
        return multiplePhrase;
    }
    return uniqValues[0];
}

export default getAggregateText;
