export const graphql = (operationName) => `api/graphql?opname=${operationName}`;

function searchObjToQuery(searchObj) {
    let result = '';
    Object.entries(searchObj).forEach(([searchCategory, searchValue], idx) => {
        result = result.concat(`${idx ? '+' : ''}${searchCategory}:`);
        if (Array.isArray(searchValue)) {
            result = result.concat(searchValue.join(','));
        } else {
            result = result.concat(searchValue);
        }
    });
    return encodeURI(result);
}

export const general = {
    graphqlOps: {
        summaryCounts: 'summary_counts',
    },
};

export const search = {
    globalSearchWithResults: '/v1/search?query=Cluster:remote',
    globalSearchWithNoResults: '/v1/search?query=Cluster:',
    options: '/v1/search/metadata/options*',
    autocomplete: 'v1/search/autocomplete*',
    autocompleteBySearch: (searchObj, category) =>
        `v1/search/autocomplete?query=${searchObjToQuery(searchObj)}&categories=${category}`,
    graphqlOps: {
        autocomplete: 'autocomplete',
    },
};

export const alerts = {
    countsByCluster: 'v1/alerts/summary/counts?*group_by=CLUSTER*',
    countsByCategory: '/v1/alerts/summary/counts?*group_by=CATEGORY*',
    alerts: '/v1/alerts?(\\?*)',
    alertById: '/v1/alerts/*',
    resolveAlert: '/v1/alerts/*/resolve',
    alertscount: '/v1/alertscount?(\\?*)',
    pageSearchAutocomplete: (searchObj) => search.autocompleteBySearch(searchObj, 'ALERTS'),
    graphqlOps: {
        getTags: 'getAlertTags',
        tagsAutocomplete: 'autocomplete',
        bulkAddAlertTags: 'bulkAddAlertTags',
        getComments: 'getAlertComments',
    },
};

export const clusters = {
    single: 'v1/clusters/**',
    list: 'v1/clusters',
    zip: 'api/extensions/clusters/zip',
};

export const risks = {
    // The * at the end exists because sometimes we add ?query= at the end.
    riskyDeployments: '/v1/deploymentswithprocessinfo*',
    riskyDeploymentsWithPagination:
        '/v1/deploymentswithprocessinfo?pagination.offset=0&pagination.limit=50&pagination.sortOption.field=Priority&pagination.sortOption.reversed=false*',
    deploymentsCount: '/v1/deploymentscount*',
    getDeployment: '/v1/deployments/*',
    fetchDeploymentWithRisk: '/v1/deploymentswithrisk/*',
    graphqlOps: {
        autocomplete: 'autocomplete',
        getProcessTags: 'getProcessTags',
        getProcessComments: 'getProcessComments',
        getDeploymentEventTimeline: 'getDeploymentEventTimeline',
        getPodEventTimeline: 'getPodEventTimeline',
    },
};

export const images = {
    list: '/v1/images*',
    count: '/v1/imagescount*',
    get: '/v1/images/*',
};

export const auth = {
    loginAuthProviders: 'v1/login/authproviders',
    authProviders: 'v1/authProviders*',
    authStatus: '/v1/auth/status',
    logout: '/sso/session/logout',
    tokenRefresh: '/sso/session/tokenrefresh',
};

export const certExpiry = {
    central: 'v1/credentialexpiry?component=CENTRAL',
    scanner: 'v1/credentialexpiry?component=SCANNER',
};

export const certGen = {
    central: 'api/extensions/certgen/central',
    scanner: 'api/extensions/certgen/scanner',
};

export const dashboard = {
    timeseries: '/v1/alerts/summary/timeseries?*',
};

export const metadata = 'v1/metadata';

export const network = {
    networkBaselineLock: '/v1/networkbaseline/**/lock',
    networkBaselineUnlock: '/v1/networkbaseline/**/unlock',
    networkBaseline: '/v1/networkbaseline/**',
    networkBaselineStatus: '/v1/networkbaseline/**/status',
    networkPoliciesGraph: '/v1/networkpolicies/cluster/*',
    networkGraph: '/v1/networkgraph/cluster/*',
    epoch: '/v1/networkpolicies/graph/epoch',
    simulate: '/v1/networkpolicies/simulate/*',
    deployment: 'v1/deployments/*',
};

export const policies = {
    policies: '/v1/policies',
    policy: '/v1/policies/*',
    dryrun: '/v1/policies/dryrunjob',
    export: '/v1/policies/export',
    import: '/v1/policies/import',
};

export const roles = {
    list: '/v1/roles/*',
    mypermissions: 'v1/mypermissions',
};

export const permissionSets = {
    list: '/v1/permissionsets',
};

export const accessScopes = {
    list: '/v1/simpleaccessscopes',
};

export const groups = {
    list: '/v1/groups/*',
};

export const userAttributes = {
    list: '/v1/userattributes/*',
};

export const compliance = {
    graphqlOps: {
        getAggregatedResults: 'getAggregatedResults',
        namespaces: 'namespaceList',
    },
    export: {
        csv: '/api/compliance/export/csv',
    },
};

export const logs = '/api/logimbue';

export const featureFlags = '/v1/featureflags';

export const configMgmt = {
    graphqlOps: {
        policies: 'policies',
        getPolicy: 'getPolicy',
    },
};

export const vulnMgmt = {
    graphqlOps: {
        getCves: 'getCves',
        getPolicies: 'getPolicies',
        getPolicy: 'getPolicy',
        getImage: 'getImage',
    },
};

export const integrationHealth = {
    imageIntegrations: '/v1/integrationhealth/imageintegrations',
    notifiers: '/v1/integrationhealth/notifiers',
    externalBackups: '/v1/integrationhealth/externalbackups',
    vulnDefinitions: '/v1/integrationhealth/vulndefinitions',
};

export const integrations = {
    imageIntegrations: '/v1/imageintegrations',
    notifiers: '/v1/notifiers',
    externalBackups: '/v1/externalbackups',
    authPlugins: '/v1/scopedaccessctrl/configs',
    apiTokens: 'v1/apitokens?revoked=false',
    clusterInitBundles: '/v1/cluster-init/init-bundles',
};

export const apiToken = {
    generate: 'v1/apitokens/generate',
};

export const system = {
    config: '/v1/config',
};

export const extensions = {
    diagnostics: '/api/extensions/diagnostics',
};

export const permissions = {
    mypermissions: '/v1/mypermissions',
};

export const apiDocs = {
    docs: '/api/docs/swagger',
};
