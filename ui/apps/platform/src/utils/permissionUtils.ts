/* eslint-disable import/prefer-default-export */
export function checkForPermissionErrorMessage(error: Error, defaultMessage?: string): string {
    if (error && typeof error.message === 'string') {
        if (error.message.includes('403') || error.message.toLowerCase().includes('permission')) {
            return 'A database error has occurred. Please check that you have the correct permissions to view this information.';
        }
        return error.message;
    }
    return defaultMessage || 'An unknown error has occurred.';
}
