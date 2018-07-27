import flatten from 'flat';
import omitBy from 'lodash/omitBy';

export default function removeEmptyFields(obj) {
    const flattenedObj = flatten(obj);
    const omittedObj = omitBy(
        flattenedObj,
        value =>
            value === null ||
            value === undefined ||
            value === '' ||
            value === [] ||
            (Array.isArray(value) && !value.length)
    );
    const newObj = flatten.unflatten(omittedObj);
    return newObj;
}
