import resolvePath from 'object-resolve-path';
import set from 'lodash/set';

import formDescriptors from 'Containers/Integrations/formDescriptors';

/**
 * Returns a field from the form descriptor for a particular integration that could
 * possibly have stored credentials
 *
 * @param {string} source - The source of the integration
 * @param {string} type - The type of the integration
 * @returns {Object}
 */
export function getFieldsWithPossibleStoredCredentials(source, type) {
    if (formDescriptors[source] && formDescriptors[source][type]) {
        const fields = formDescriptors[source][type];
        const fieldsWithStoredCredentials = fields.filter(
            (field) => 'checkStoredCredentials' in field
        );
        return fieldsWithStoredCredentials;
    }
    return [];
}

/**
 * If the form field is filled for a password, where the integration could possibly have
 * stored credentials, then return true, otherwise, return false
 *
 * @param {string} source - The source of the integration
 * @param {string} type - The type of the integration
 * @param {Object} data - The form data
 * @returns {boolean}
 */
function shouldUpdateStoredCredentials(source, type, data) {
    const fieldsWithPossibleStoredCredentials = getFieldsWithPossibleStoredCredentials(
        source,
        type
    );
    if (fieldsWithPossibleStoredCredentials.length === 0) {
        return false;
    }
    const shouldUpdate = fieldsWithPossibleStoredCredentials.some((field) => {
        return !!resolvePath(data, field.jsonpath);
    });
    return shouldUpdate;
}

/**
 * Takes the initial form data, and if the integration has a password value of "******", it'll
 * return that field from the form descriptor
 *
 * @param {string} source - The source of the integration
 * @param {string} type - The type of the integration
 * @param {Object} data - The form data
 * @returns {Object}
 */
function findFieldsWithStoredCredentials(source, type, data) {
    const fieldsWithPossibleStoredCredentials = getFieldsWithPossibleStoredCredentials(
        source,
        type
    );
    if (fieldsWithPossibleStoredCredentials.length === 0) {
        return [];
    }
    const fieldsWithStoredCredentials = fieldsWithPossibleStoredCredentials.filter((field) => {
        const value = resolvePath(data, field.jsonpath);
        return value === '******';
    });
    return fieldsWithStoredCredentials;
}

/**
 * Takes the initial form data, and adds a new field called "hasStoredCredentials", if the
 * integration has a password value of "******" (something backend decided would determine if
 * it currently has credentials stored in memory)
 *
 * @param {string} source - The source of the integration
 * @param {string} type - The type of the integration
 * @param {Object} data - The form data
 * @returns {Object}
 */
export function setStoredCredentialFields(source, type, initialValues) {
    const fieldsWithStoredCredentials = findFieldsWithStoredCredentials(
        source,
        type,
        initialValues
    );
    // if there isn't a field that uses the stored credentials, leave the data untouched
    if (fieldsWithStoredCredentials.length === 0) {
        return initialValues;
    }
    const newInitialValues = { ...initialValues };
    newInitialValues.hasStoredCredentials = true;
    fieldsWithStoredCredentials.forEach((field) => {
        set(newInitialValues, field.jsonpath, '');
    });
    return newInitialValues;
}

/**
 * Determines if an integration can possibly use stored credentials, and if it does,
 * it'll set an options object with the "updatePassword" set to the appropriate value
 *
 * @param {string} source - The source of the integration
 * @param {string} type - The type of the integration
 * @param {Object} data - The form data
 * @param {Object} metadata - Extra information used to determine how to set options (e.g. "isNewIntegration")
 * @returns {Object}
 */
export function setFormSubmissionOptions(source, type, data, metadata = {}) {
    const fieldsWithPossibleStoredCredentials = getFieldsWithPossibleStoredCredentials(
        source,
        type
    );
    let options = null;
    if (fieldsWithPossibleStoredCredentials.length) {
        const { isNewIntegration } = metadata;
        // if we're creating a new integration for something that can store credentials, we should
        // automatically update
        if (isNewIntegration) {
            return { updatePassword: true };
        }
        const updatePassword = shouldUpdateStoredCredentials(source, type, data);
        options = { updatePassword };
    }
    return options;
}
