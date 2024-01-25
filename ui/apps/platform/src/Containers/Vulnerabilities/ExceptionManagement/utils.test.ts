import { ExceptionStatus, VulnerabilityException } from 'services/VulnerabilityExceptionService';
import { getImageScopeSearchValue, getVulnerabilityState } from './utils';

describe('ExceptionManagement utils', () => {
    describe('imageScopeSearchValue', () => {
        it('should return empty string for all .*', () => {
            const imageScope = { registry: '.*', remote: '.*', tag: '.*' };
            const searchValue = getImageScopeSearchValue({ imageScope });

            expect(searchValue).toEqual('');
        });

        it('should return registry/remote for .* tag', () => {
            const imageScope = { registry: 'registry', remote: 'remote', tag: '.*' };
            const searchValue = getImageScopeSearchValue({ imageScope });

            expect(searchValue).toEqual('registry/remote');
        });

        it('should return registry/remote:tag for non .* tag', () => {
            const imageScope = { registry: 'registry', remote: 'remote', tag: 'tag' };
            const searchValue = getImageScopeSearchValue({ imageScope });

            expect(searchValue).toEqual('registry/remote:tag');
        });
    });

    describe('getVulnerabilityState', () => {
        function makeVulnerabilityException(
            status: ExceptionStatus,
            type: 'DEFERRAL' | 'FALSE_POSITIVE'
        ): VulnerabilityException {
            const exception = {
                id: '123',
                cves: ['CVE-123-456'],
                comments: [],
                status,
                scope: {
                    imageScope: {
                        registry: 'registry',
                        remote: 'remote',
                        tag: 'tag',
                    },
                },
                createdAt: '2021-01-01T00:00:00.000Z',
                name: 'Exception Name',
                expired: false,
                requester: { id: '123', name: 'username' },
                approvers: [{ id: '123', name: 'Alice' }],
                lastUpdated: '2021-01-01T00:00:00.000Z',
            };

            if (type === 'DEFERRAL') {
                return {
                    ...exception,
                    targetState: 'DEFERRED',
                    deferralRequest: {
                        expiry: {
                            expiryType: 'TIME',
                            expiresOn: '2021-01-01T00:00:00.000Z',
                        },
                    },
                };
            }

            return {
                ...exception,
                targetState: 'FALSE_POSITIVE',
                falsePositiveRequest: {},
            };
        }

        it('should return "OBSERVED" for unapproved exceptions', () => {
            const pendingDeferral = makeVulnerabilityException('PENDING', 'DEFERRAL');
            const deniedDeferral = makeVulnerabilityException('DENIED', 'DEFERRAL');
            const pendingFalsePositive = makeVulnerabilityException('PENDING', 'FALSE_POSITIVE');
            const deniedFalsePositive = makeVulnerabilityException('DENIED', 'FALSE_POSITIVE');

            expect(getVulnerabilityState(deniedDeferral)).toEqual('OBSERVED');
            expect(getVulnerabilityState(pendingDeferral)).toEqual('OBSERVED');
            expect(getVulnerabilityState(deniedFalsePositive)).toEqual('OBSERVED');
            expect(getVulnerabilityState(pendingFalsePositive)).toEqual('OBSERVED');
        });

        it('should return "DEFERRED" for approved deferrals', () => {
            const approvedDeferral = makeVulnerabilityException('APPROVED', 'DEFERRAL');
            const approvedPendingUpdateDeferral = makeVulnerabilityException(
                'APPROVED_PENDING_UPDATE',
                'DEFERRAL'
            );

            expect(getVulnerabilityState(approvedDeferral)).toEqual('DEFERRED');
            expect(getVulnerabilityState(approvedPendingUpdateDeferral)).toEqual('DEFERRED');
        });

        it('should return "FALSE_POSITIVE" for approved false positives', () => {
            const approvedFalsePositive = makeVulnerabilityException('APPROVED', 'FALSE_POSITIVE');
            const approvedPendingUpdateFalsePositive = makeVulnerabilityException(
                'APPROVED_PENDING_UPDATE',
                'FALSE_POSITIVE'
            );

            expect(getVulnerabilityState(approvedFalsePositive)).toEqual('FALSE_POSITIVE');
            expect(getVulnerabilityState(approvedPendingUpdateFalsePositive)).toEqual(
                'FALSE_POSITIVE'
            );
        });
    });
});
