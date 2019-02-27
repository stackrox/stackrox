export const alerts = {
    countsByCluster: 'v1/alerts/summary/counts?*group_by=CLUSTER*',
    countsByCategory: '/v1/alerts/summary/counts?*group_by=CATEGORY*',
    alerts: '/v1/alerts?*',
    alertById: '/v1/alerts/*'
};

export const clusters = {
    list: 'v1/clusters',
    zip: 'api/extensions/clusters/zip'
};

export const risks = {
    riskyDeployments: 'v1/deployments*'
};

export const search = {
    globalSearchWithResults: '/v1/search?query=Cluster:remote',
    globalSearchWithNoResults: '/v1/search?query=Cluster:',
    options: '/v1/search/metadata/options*'
};

export const images = {
    list: '/v1/images*'
};

export const auth = {
    authProviders: 'v1/authProviders*',
    authStatus: '/v1/auth/status'
};

export const dashboard = {
    timeseries: '/v1/alerts/summary/timeseries?*'
};

export const network = {
    networkGraph: '/v1/networkpolicies/cluster/*',
    epoch: '/v1/networkpolicies/graph/epoch'
};

export const policies = {
    policy: 'v1/policies/*',
    dryrun: 'v1/policies/dryrun'
};

export const roles = {
    list: '/v1/roles/*'
};

export const summary = {
    counts: '/v1/summary/counts'
};

export const compliance = {
    export: {
        csv: '/api/compliance/export/csv'
    }
};
