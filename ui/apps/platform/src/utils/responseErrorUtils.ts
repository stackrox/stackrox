import type { AxiosError } from 'axios';
import { ApolloError } from '@apollo/client';
import upperFirst from 'lodash/upperFirst';

function isAxiosError(error: Error): error is AxiosError<{ message?: string }> {
    return (
        Object.prototype.hasOwnProperty.call(error, 'response') ||
        Object.prototype.hasOwnProperty.call(error, 'request')
    );
}

const commonStatusCodeNameMap = {
    401: 'Unauthorized',
    403: 'Forbidden',
    404: 'Not Found',
    500: 'Internal Server Error',
    501: 'Not Implemented',
    502: 'Bad Gateway',
    503: 'Service Unavailable',
    504: 'Gateway Timeout',
} as const;

/*
 * Given argument of promise-catch method or try-catch block for an axios call,
 * return the error message for display.
 *
 * Backend errors follow Go conventions (lower-case first letter); capitalize the
 * first character for display. upperFirst (not capitalize) preserves the rest of
 * the message verbatim, e.g. names and quotes in "no connection to cluster ...".
 *
 * For classifying an error by matching case-sensitive patterns against its text,
 * use getRawAxiosErrorMessage instead, so the capitalization does not defeat a
 * pattern anchored on a lower-case leading word.
 */
export function getAxiosErrorMessage(error: unknown): string {
    return upperFirst(getRawAxiosErrorMessage(error));
}

/*
 * Same as getAxiosErrorMessage but without display capitalization, returning the
 * error text verbatim. Use it for case-sensitive matching, not for display.
 */
export function getRawAxiosErrorMessage(error: unknown): string {
    // See https://axios-http.com/docs/handling_errors

    if (error instanceof Error) {
        // Handle network errors from failed GraphQL requests
        if (
            error instanceof ApolloError &&
            error.networkError &&
            'result' in error.networkError &&
            typeof error.networkError.result === 'string'
        ) {
            // Display a user-friendly error message for common HTTP status codes, falling back to
            // the error name for less common codes
            const name =
                commonStatusCodeNameMap[error.networkError.statusCode] ?? error.networkError.name;
            return `${name}: ${error.networkError.result}`;
        }

        if (isAxiosError(error)) {
            if (error.response?.status === 403) {
                return 'Please check that your role has the required permissions.';
            }

            if (typeof error.response?.data?.message === 'string') {
                // The server responded to the request with an error.
                return error.response.data.message;
            }

            if (error.request instanceof XMLHttpRequest) {
                // No response was received for the request.
                return error.request.statusText || error.message;
            }
        }

        // An error was thrown before the request.
        return error.message;
    }

    return 'Unknown error';
}

export function isTimeoutError(error: unknown): boolean {
    if (error instanceof Error && isAxiosError(error)) {
        return error.code === 'ECONNABORTED';
    }
    return false;
}
