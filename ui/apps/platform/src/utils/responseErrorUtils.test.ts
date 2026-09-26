import { getAxiosErrorMessage } from './responseErrorUtils';

describe('responseErrorUtils', () => {
    describe('getAxiosErrorMessage', () => {
        // The remainder deliberately contains upper-case characters (a cluster name and
        // acronym) so these tests fail if capitalization regresses to lodash/capitalize,
        // which lower-cases everything after the first character.
        it('capitalizes the first character while preserving the remainder of a plain error', () => {
            const error = new Error('no connection to cluster "Prod-East" (OCP)');

            expect(getAxiosErrorMessage(error)).toBe('No connection to cluster "Prod-East" (OCP)');
        });

        it('capitalizes a backend message returned in the axios response body', () => {
            const error = Object.assign(new Error('Request failed'), {
                response: {
                    data: { message: 'cluster "Prod-East" does not support delegated scanning' },
                },
            });

            expect(getAxiosErrorMessage(error)).toBe(
                'Cluster "Prod-East" does not support delegated scanning'
            );
        });

        it('leaves an already-capitalized message unchanged', () => {
            const error = new Error('Unable to reach Cluster "Prod-East"');

            expect(getAxiosErrorMessage(error)).toBe('Unable to reach Cluster "Prod-East"');
        });

        it('returns a fallback for non-error input', () => {
            expect(getAxiosErrorMessage(undefined)).toBe('Unknown error');
        });
    });
});
