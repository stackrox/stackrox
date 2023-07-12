import isString from 'lodash/isString';
import isPlainObject from 'lodash/isPlainObject';
import isFunction from 'lodash/isFunction';
import mapValues from 'lodash/mapValues';

function appendPrefixDeep(prefix, selector) {
    if (isPlainObject(selector)) {
        return mapValues(selector, (val) => appendPrefixDeep(prefix, val));
    }
    if (isString(selector)) {
        return `${prefix} ${selector}`;
    }
    if (isFunction(selector)) {
        return (...args) => appendPrefixDeep(prefix, selector(...args));
    }
    throw new Error(`Unexpected type of selector ${JSON.stringify(selector)}`);
}

/**
 * Utility function to define selectors within container without boilerplate of prefixing them manually.
 * @example
 * // returns
 * // {
 * //   btn: 'container button',
 * //   innerContainer: {
 * //     label: 'container inner-container label'
 * //   },
 * //   dynamic: val => `container ${val}`
 * // }
 * scopeSelectors('container', {
 *   btn: 'button',
 *   innerContainer: scopeSelectors('inner-container', {
 *     label: 'label'
 *   }),
 *   dynamic: val => `${val}`
 * });
 *
 * @template {Object} T
 * @param {string} containerSelector selector of the container that will be added to all selectors within it
 * @param {T} selectorsWithin selectors within the container to be prefixed
 * @returns {T} prefixed selectors within the container
 */
export default function scopeSelectors(containerSelector, selectorsWithin) {
    return appendPrefixDeep(containerSelector, selectorsWithin);
}
