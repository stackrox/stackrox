// system under test (SUT)
import { checkForPermissionErrorMessage } from './permissionUtils';

describe('permissionUtils', () => {
    describe('checkForPermissionErrorMessage', () => {
        it('should return a default error when no message property in argument', () => {
            const error = { name: 'Error' };

            const errorMessage = checkForPermissionErrorMessage(error);

            expect(errorMessage).toEqual('An unknown error has occurred.');
        });

        it('should return the supplied default error when no message property in argument', () => {
            const error = { name: 'Error' };
            const defaultMessage = `An error occurred in retrieving deployments. Please refresh the page. If this problem continues, please contact support.`;

            const errorMessage = checkForPermissionErrorMessage(error, defaultMessage);

            expect(errorMessage).toEqual(defaultMessage);
        });

        it('should return message property in argument', () => {
            const error = new Error('Network error');

            const errorMessage = checkForPermissionErrorMessage(error);

            expect(errorMessage).toEqual('Network error');
        });

        it('should return a permissions message if error message contains the word permission', () => {
            const error = new Error(
                'rpc error: code = PermissionDenied desc = not authorized: "READ_ACCESS" for "Compliance"'
            );

            const errorMessage = checkForPermissionErrorMessage(error);

            expect(errorMessage).toEqual(
                'A database error has occurred. Please check that you have the correct permissions to view this information.'
            );
        });

        it('should return a permissions message if error message contains the number 403', () => {
            const error = new Error('Request failed with status code 403');

            const errorMessage = checkForPermissionErrorMessage(error);

            expect(errorMessage).toEqual(
                'A database error has occurred. Please check that you have the correct permissions to view this information.'
            );
        });
    });
});
