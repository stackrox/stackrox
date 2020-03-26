import logError from './logError';

/**
 * @typedef {Object} Result
 * @property {boolean} hasErrors - Tells you if there was at least one error
 * @property {string[]} errorMessages - The list of error messages
 */

/**
 * Captures each error exception, in an array of exceptions, to log server-side and returns
 * the necessary information about the errors
 * @param {Object[]} errors - The GraphQL query/mutation errors
 * @returns {Result}
 */
function captureGraphQLErrors(errors) {
    const errorMessages = [];
    errors.forEach(error => {
        if (error) {
            logError(error);
            errorMessages.push(error.message);
        }
    });
    return {
        hasErrors: !!errorMessages.length,
        errorMessages
    };
}

export default captureGraphQLErrors;
