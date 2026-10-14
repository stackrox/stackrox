import { getAxiosErrorMessage } from './responseErrorUtils';

describe('responseErrorUtils', () => {
    describe('getAxiosErrorMessage', () => {
        it('preserves a plain error message including its original casing', () => {
            const error = new Error('no connection to cluster "Prod-East" (OCP)');

            expect(getAxiosErrorMessage(error)).toBe('no connection to cluster "Prod-East" (OCP)');
        });

        it('preserves a backend message returned in the axios response body', () => {
            const error = Object.assign(new Error('Request failed'), {
                response: {
                    data: { message: 'cluster "Prod-East" does not support delegated scanning' },
                },
            });

            expect(getAxiosErrorMessage(error)).toBe(
                'cluster "Prod-East" does not support delegated scanning'
            );
        });

        it('preserves a message beginning with a lower-case acronym', () => {
            const error = new Error('gRPC connection to Cluster "Prod-East" failed');

            expect(getAxiosErrorMessage(error)).toBe(
                'gRPC connection to Cluster "Prod-East" failed'
            );
        });

        it('returns a fallback for non-error input', () => {
            expect(getAxiosErrorMessage(undefined)).toBe('Unknown error');
        });
    });
});
