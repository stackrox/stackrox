import { AxiosError } from 'axios';

function isAxiosError(error: Error): error is AxiosError {
    return (
        Object.prototype.hasOwnProperty.call(error, 'response') ||
        Object.prototype.hasOwnProperty.call(error, 'request')
    );
}

/*
 * Given argument of promise-catch method or try-catch block for an axios call,
 * return error message.
 */
export function getAxiosErrorMessage(error: unknown): string {
    // See https://axios-http.com/docs/handling_errors

    if (error instanceof Error) {
        if (isAxiosError(error)) {
            if (error.response?.status === 403) {
                return 'Please check that your role has the required permissions.';
            }

            if (typeof error.response?.data?.message === 'string') {
                // The server responded to the request with an error.
                // eslint-disable-next-line @typescript-eslint/no-unsafe-return
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
