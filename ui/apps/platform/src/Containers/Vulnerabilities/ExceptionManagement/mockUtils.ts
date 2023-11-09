import { VulnerabilityException } from 'services/VulnerabilityExceptionService';

export const vulnerabilityExceptions: VulnerabilityException[] = [
    {
        id: 'ecf6b94a-d857-4e69-b50f-cda1cdf002ef',
        name: 'ecf6b94a-d857-4e69-b50f-cda1cdf002ef',
        targetState: 'DEFERRED',
        status: 'APPROVED_PENDING_UPDATE',
        expired: false,
        requester: {
            id: 'sso:4df1b98c-24ed-4073-a9ad-356aec6bb62d:admin',
            name: 'admin',
        },
        approvers: [],
        createdAt: '2023-10-01T19:16:49.155480945Z',
        lastUpdated: '2023-10-01T19:16:49.155480945Z',
        comments: [
            {
                createdAt: '2023-10-23T19:16:49.155480945Z',
                id: 'c84b3f5f-4cad-4c4e-8a4a-97b821c2c373',
                message: 'Defer me!',
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
        cves: ['CVE-2018-20839', 'RHSA-2023:5997'],
    },
    {
        id: '5837bb34-5357-4b78-ad2b-188fc0b33e78',
        name: '5837bb34-5357-4b78-ad2b-188fc0b33e78',
        targetState: 'FALSE_POSITIVE',
        status: 'APPROVED_PENDING_UPDATE',
        expired: false,
        requester: {
            id: 'sso:4df1b98c-24ed-4073-a9ad-356aec6bb62d:admin',
            name: 'admin',
        },
        approvers: [],
        createdAt: '2023-10-01T19:16:49.155480945Z',
        lastUpdated: '2023-10-01T19:16:49.155480945Z',
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
        falsePositiveRequest: {},
        falsePositiveUpdate: {
            cves: ['CVE-2020-20839'],
        },
        cves: ['CVE-2020-20839', 'CVE-2020-20840'],
    },
    {
        id: '6837bb34-5357-4b78-ad2b-188fc0b33e78',
        name: '6837bb34-5357-4b78-ad2b-188fc0b33e78',
        targetState: 'DEFERRED',
        status: 'DENIED',
        expired: false,
        requester: {
            id: 'sso:4df1b98c-24ed-4073-a9ad-356aec6bb62d:admin',
            name: 'admin',
        },
        approvers: [],
        createdAt: '2023-10-01T19:16:49.155480945Z',
        lastUpdated: '2023-10-01T19:16:49.155480945Z',
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
        deferralRequest: {
            expiry: {
                expiryType: 'ALL_CVE_FIXABLE',
            },
        },
        cves: ['CVE-2018-20839', 'CVE-2018-20840'],
    },
    {
        id: '7837bb34-5357-4b78-ad2b-188fc0b33e78',
        name: '7837bb34-5357-4b78-ad2b-188fc0b33e78',
        targetState: 'FALSE_POSITIVE',
        status: 'DENIED',
        expired: false,
        requester: {
            id: 'sso:4df1b98c-24ed-4073-a9ad-356aec6bb62d:admin',
            name: 'admin',
        },
        approvers: [],
        createdAt: '2023-10-01T19:16:49.155480945Z',
        lastUpdated: '2023-10-01T19:16:49.155480945Z',
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
        falsePositiveRequest: {},
        cves: ['CVE-2020-20839', 'CVE-2020-20840'],
    },
];

export const pendingRequests = vulnerabilityExceptions.filter(
    (exception) => exception.status === 'PENDING' || exception.status === 'APPROVED_PENDING_UPDATE'
);

export const approvedDeferrals = vulnerabilityExceptions.filter(
    (exception) =>
        exception.targetState === 'DEFERRED' &&
        (exception.status === 'APPROVED' || exception.status === 'APPROVED_PENDING_UPDATE')
);

export const approvedFalsePositives = vulnerabilityExceptions.filter(
    (exception) =>
        exception.targetState === 'FALSE_POSITIVE' &&
        (exception.status === 'APPROVED' || exception.status === 'APPROVED_PENDING_UPDATE')
);

export const deniedRequests = vulnerabilityExceptions.filter(
    (exception) => exception.status === 'DENIED'
);
