import {
    BaseVulnerabilityException,
    VulnerabilityException,
} from 'services/VulnerabilityExceptionService';
import {
    getShouldUseUpdatedRequest,
    getRequestedAction,
    RequestContext,
} from './ExceptionRequestTableCells';

const baseException: BaseVulnerabilityException = {
    id: '4837bb34-5357-4b78-ad2b-188fc0b33e78',
    name: '4837bb34-5357-4b78-ad2b-188fc0b33e78',
    status: 'APPROVED_PENDING_UPDATE',
    expired: false,
    requester: {
        id: 'sso:4df1b98c-24ed-4073-a9ad-356aec6bb62d:admin',
        name: 'admin',
    },
    approvers: [],
    createdAt: '2023-10-01T19:16:48.155480945Z',
    lastUpdated: '2023-10-01T19:16:48.155480945Z',
    comments: [
        {
            createdAt: '2023-10-23T19:16:49.155480945Z',
            id: 'c84b3f5f-4cad-4c4e-8a4a-97b821c2c373',
            message: 'asdf',
            user: {
                id: 'sso:4df1b98c-24ed-4073-a9ad-356aec6bb62d:admin',
                name: 'admin',
            },
        },
    ],
    scope: {
        imageScope: {
            registry: 'quay.io',
            remote: 'stackrox-io/scanner',
            tag: '.*',
        },
    },
    cves: ['CVE-2018-20839'],
};

describe('ExceptionRequestTableCells', () => {
    describe('getDeferralExpiryToUse', () => {
        it('should use the original expiry for a pending request', () => {
            const vulnerabilityException: VulnerabilityException = {
                ...baseException,
                targetState: 'DEFERRED',
                status: 'PENDING',
                deferralRequest: {
                    expiry: {
                        expiryType: 'ALL_CVE_FIXABLE',
                    },
                },
            };
            const context: RequestContext = 'PENDING_UPDATE';

            const shouldUseUpdatedRequest = getShouldUseUpdatedRequest(
                vulnerabilityException,
                context
            );

            expect(shouldUseUpdatedRequest).toBe(false);
        });

        // When an approved deferral that is pending an update is finally approved, it overwrites
        // the deferralReq with the deferralUpdate
        it('should show the original expiry for an approved request', () => {
            const vulnerabilityException: VulnerabilityException = {
                ...baseException,
                targetState: 'DEFERRED',
                status: 'APPROVED',
                deferralRequest: {
                    expiry: {
                        expiryType: 'ALL_CVE_FIXABLE',
                    },
                },
            };
            const context: RequestContext = 'PENDING_UPDATE';

            const shouldUseUpdatedRequest = getShouldUseUpdatedRequest(
                vulnerabilityException,
                context
            );

            expect(shouldUseUpdatedRequest).toBe(false);
        });

        it('should use the updated expiry for an approved request pending an update', () => {
            const vulnerabilityException: VulnerabilityException = {
                ...baseException,
                targetState: 'DEFERRED',
                status: 'APPROVED_PENDING_UPDATE',
                deferralRequest: {
                    expiry: {
                        expiryType: 'ALL_CVE_FIXABLE',
                    },
                },
                deferralUpdate: {
                    cves: ['CVE-2018-20839'],
                    expiry: {
                        expiryType: 'TIME',
                        expiresOn: '2023-10-31T19:16:49.155480945Z',
                    },
                },
            };
            const context: RequestContext = 'PENDING_UPDATE';

            const shouldUseUpdatedRequest = getShouldUseUpdatedRequest(
                vulnerabilityException,
                context
            );

            expect(shouldUseUpdatedRequest).toBe(true);
        });

        it('should use the original expiry for an approved request pending an update', () => {
            const vulnerabilityException: VulnerabilityException = {
                ...baseException,
                targetState: 'DEFERRED',
                status: 'APPROVED_PENDING_UPDATE',
                deferralRequest: {
                    expiry: {
                        expiryType: 'ALL_CVE_FIXABLE',
                    },
                },
                deferralUpdate: {
                    cves: ['CVE-2018-20839'],
                    expiry: {
                        expiryType: 'TIME',
                        expiresOn: '2023-10-31T19:16:49.155480945Z',
                    },
                },
            };
            const context: RequestContext = 'CURRENT';

            const shouldUseUpdatedRequest = getShouldUseUpdatedRequest(
                vulnerabilityException,
                context
            );

            expect(shouldUseUpdatedRequest).toBe(false);
        });
    });

    describe('getRequestedAction', () => {
        it('should show the requested action for a pending false positive request', () => {
            const vulnerabilityException: VulnerabilityException = {
                ...baseException,
                targetState: 'FALSE_POSITIVE',
                status: 'PENDING',
                falsePositiveRequest: {},
            };
            const context: RequestContext = 'PENDING_UPDATE';

            const requestedAction = getRequestedAction(vulnerabilityException, context);

            expect(requestedAction).toBe('False positive');
        });

        it('should show the requested action for a pending deferral with CVEs deferred until all fixed', () => {
            const vulnerabilityException: VulnerabilityException = {
                ...baseException,
                targetState: 'DEFERRED',
                status: 'PENDING',
                deferralRequest: {
                    expiry: {
                        expiryType: 'ALL_CVE_FIXABLE',
                    },
                },
            };
            const context: RequestContext = 'PENDING_UPDATE';

            const requestedAction = getRequestedAction(vulnerabilityException, context);

            expect(requestedAction).toBe('Deferred (when all fixed)');
        });

        it('should show the requested action for a pending deferral with CVEs deferred until one fixed', () => {
            const vulnerabilityException: VulnerabilityException = {
                ...baseException,
                targetState: 'DEFERRED',
                status: 'PENDING',
                deferralRequest: {
                    expiry: {
                        expiryType: 'ANY_CVE_FIXABLE',
                    },
                },
            };
            const context: RequestContext = 'PENDING_UPDATE';

            const requestedAction = getRequestedAction(vulnerabilityException, context);

            expect(requestedAction).toBe('Deferred (when any fixed)');
        });

        it('should show the requested action for a pending deferral with CVEs deferred for 30 days', () => {
            const vulnerabilityException: VulnerabilityException = {
                ...baseException,
                targetState: 'DEFERRED',
                status: 'PENDING',
                deferralRequest: {
                    expiry: {
                        expiryType: 'TIME',
                        expiresOn: '2023-10-31T19:16:49.155480945Z',
                    },
                },
            };
            const context: RequestContext = 'PENDING_UPDATE';

            const requestedAction = getRequestedAction(vulnerabilityException, context);

            expect(requestedAction).toBe('Deferred (in 30 days)');
        });

        it('should show the requested action for a pending deferral with CVEs deferred indefinitely', () => {
            const vulnerabilityException: VulnerabilityException = {
                ...baseException,
                targetState: 'DEFERRED',
                status: 'PENDING',
                deferralRequest: {
                    expiry: {
                        expiryType: 'TIME',
                        expiresOn: null,
                    },
                },
            };
            const context: RequestContext = 'PENDING_UPDATE';

            const requestedAction = getRequestedAction(vulnerabilityException, context);

            expect(requestedAction).toBe('Deferred (indefinitely)');
        });

        it('should show the requested action for an approved deferral (pending update) with CVEs deferred for 30 days', () => {
            const vulnerabilityException: VulnerabilityException = {
                ...baseException,
                targetState: 'DEFERRED',
                status: 'APPROVED_PENDING_UPDATE',
                deferralRequest: {
                    expiry: {
                        expiryType: 'TIME',
                        expiresOn: null,
                    },
                },
                deferralUpdate: {
                    cves: ['CVE-2018-20839'],
                    expiry: {
                        expiryType: 'TIME',
                        expiresOn: '2023-10-31T19:16:49.155480945Z',
                    },
                },
            };
            const context: RequestContext = 'PENDING_UPDATE';

            const requestedAction = getRequestedAction(vulnerabilityException, context);

            expect(requestedAction).toBe('Deferral pending update (in 30 days)');
        });

        it('should show the requested action for an approved deferral (pending update) with CVEs deferred indefinitely', () => {
            const vulnerabilityException: VulnerabilityException = {
                ...baseException,
                targetState: 'DEFERRED',
                status: 'APPROVED_PENDING_UPDATE',
                deferralRequest: {
                    expiry: {
                        expiryType: 'TIME',
                        expiresOn: null,
                    },
                },
                deferralUpdate: {
                    cves: ['CVE-2018-20839'],
                    expiry: {
                        expiryType: 'TIME',
                        expiresOn: '2023-10-31T19:16:49.155480945Z',
                    },
                },
            };
            const context: RequestContext = 'CURRENT';

            const requestedAction = getRequestedAction(vulnerabilityException, context);

            expect(requestedAction).toBe('Deferred (indefinitely)');
        });
    });
});
