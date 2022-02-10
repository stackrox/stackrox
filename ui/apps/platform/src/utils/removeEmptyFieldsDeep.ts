import cloneDeep from 'lodash/cloneDeep';
import isEmpty from 'lodash/isEmpty';
import isNil from 'lodash/isNil';
import isObject from 'lodash/isObject';
import isPlainObject from 'lodash/isPlainObject';
import mapValues from 'lodash/mapValues';
import pickBy from 'lodash/pickBy';

/**
 * Checks whether the value is empty (null, undefined, empty string, empty array, empty plain object).
 */
const isNilOrEmpty = (v: unknown): boolean => isNil(v) || v === '' || (isObject(v) && isEmpty(v));

/**
 * Removes empty fields from the object traversing deep into fields with object values.
 *
 * @param object any plain object, it'll not be mutated.
 * @param {EmptyValuePredicate} [predicate=isNilOrEmpty] either a given field value is empty
 * @returns returns a deep copy of the original object with empty fields removed
 */
export default function removeEmptyFieldsDeep(
    obj: Record<string, unknown>
): Record<string, unknown> {
    const cloned = cloneDeep(obj);
    // deep clean all the fields with values being objects themselves
    const onlyCleanNestedObjects = mapValues(pickBy(cloned, isPlainObject), removeEmptyFieldsDeep);
    // return back fields with non-object values
    const allFields = {
        ...onlyCleanNestedObjects,
        ...pickBy(cloned, (v) => !isPlainObject(v)),
    };
    // filter out empty fields
    return pickBy(allFields, (v) => !isNilOrEmpty(v));
}
