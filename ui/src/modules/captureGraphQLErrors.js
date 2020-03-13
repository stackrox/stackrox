import logError from './logError';

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
