// system under test (SUT)
import { checkForPermissionErrorMessage } from './permissionUtils';

describe('permissionUtils', () => {
    describe('checkForPermissionErrorMessage', () => {
        it('should return unknown message when no message property in error argument', () => {
            const error = { name: 'Error' }; // no message property implies a backend problem

            const errorMessage = checkForPermissionErrorMessage(error);

            expect(errorMessage).toEqual('An unknown error has occurred.');
        });

        it('should return unknown message when no message property in error argument even if defaultMessage argument', () => {
            const error = { name: 'Error' }; // no message property implies a backend problem
            const defaultMessage = `An error occurred in retrieving deployments. Please refresh the page. If this problem continues, please contact support.`;

            const errorMessage = checkForPermissionErrorMessage(error, defaultMessage);

            expect(errorMessage).toEqual('An unknown error has occurred.');
        });

        it('should return defaultMessage argument if message property in error argument', () => {
            const error = { name: 'Error', message: '404 Not Found' };
            const defaultMessage = `An error occurred in retrieving deployments. Please refresh the page. If this problem continues, please contact support.`;

            const errorMessage = checkForPermissionErrorMessage(error, defaultMessage);

            expect(errorMessage).toEqual(defaultMessage);
        });

        it('should return message property in error argument if no defaultMessage argument', () => {
            const error = new Error('Network error');

            const errorMessage = checkForPermissionErrorMessage(error);

            expect(errorMessage).toEqual('Network error');
        });

        it('should return permissions message if error message includes lowercase "not authorized:"', () => {
            const error = new Error('not authorized: "READ_ACCESS" for "Compliance"');

            const errorMessage = checkForPermissionErrorMessage(error);

            expect(errorMessage).toEqual(
                'A database error has occurred. Please check that you have the correct permissions to view this information.'
            );
        });

        it('should return permissions message if error message includes sentence case "Not authorized:"', () => {
            const error = new Error('Not authorized: "READ_ACCESS" for "Compliance"');

            const errorMessage = checkForPermissionErrorMessage(error);

            expect(errorMessage).toEqual(
                'A database error has occurred. Please check that you have the correct permissions to view this information.'
            );
        });

        it('should return permissions message if error message includes "403"', () => {
            const error = new Error('Request failed with status code 403');

            const errorMessage = checkForPermissionErrorMessage(error);

            expect(errorMessage).toEqual(
                'A database error has occurred. Please check that you have the correct permissions to view this information.'
            );
        });
    });
});
