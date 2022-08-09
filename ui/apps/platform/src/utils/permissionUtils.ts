export function checkForPermissionErrorMessage(error: Error, defaultMessage?: string): string {
    if (typeof error?.message === 'string') {
        if (error.message.includes('403') || error.message.includes('not authorized')) {
            return 'A database error has occurred. Please check that you have the correct permissions to view this information.';
        }
        if (defaultMessage) {
            return defaultMessage;
        }
        return error.message;
    }
    return 'An unknown error has occurred.';
}
