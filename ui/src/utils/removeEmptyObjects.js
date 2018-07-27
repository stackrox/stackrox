import flatten from 'flat';
import omitBy from 'lodash/omitBy';
import isEmpty from 'lodash/isEmpty';

export default function removeEmptyObjects(obj) {
    const flattenedObj = flatten(obj);
    const omittedObj = omitBy(flattenedObj, value => isEmpty(value));
    const newObj = flatten.unflatten(omittedObj);
    return newObj;
}
