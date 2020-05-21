export const SEARCH_CATEGORIES = {
    ALERTS: 'ALERTS',
    DEPLOYMENTS: 'DEPLOYMENTS',
    IMAGES: 'IMAGES',
    POLICIES: 'POLICIES',
    PROCESS_INDICATORS: 'PROCESS_INDICATORS',
    SEARCH_UNSET: 'SEARCH_UNSET',
    SECRETS: 'SECRETS',
    COMPLIANCE: 'COMPLIANCE',
    SUBJECT: 'SUBJECT',
};

export const CLIENT_SIDE_SEARCH_OPTIONS = {
    COMPLIANCE: {
        STATE: 'Compliance State',
    },
    POLICY_STATUS: {
        CATEGORY: 'Policy Status',
        VALUES: {
            PASS: 'Pass',
            FAIL: 'Fail',
        },
    },
};

export const availableSearchOptions = Object.values(SEARCH_CATEGORIES);
