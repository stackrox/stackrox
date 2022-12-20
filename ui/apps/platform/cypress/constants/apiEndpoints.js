export const graphql = (opname) => `/api/graphql?opname=${opname}`;

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

export const search = {
    results: '/v1/search?query=*',
    options: '/v1/search/metadata/options*',
    optionsCategories: (categories) => `/v1/search/metadata/options?categories=${categories}`,
    autocomplete: 'v1/search/autocomplete*',
    autocompleteBySearch: (searchObj, category) =>
        `v1/search/autocomplete?query=${searchObjToQuery(searchObj)}&categories=${category}`,
    graphqlOps: {
        autocomplete: 'autocomplete',
    },
};

export const clusters = {
    list: 'v1/clusters',
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
        getDeploymentEventTimeline: 'getDeploymentEventTimeline',
        getPodEventTimeline: 'getPodEventTimeline',
    },
};

export const auth = {
    availableAuthProviders: '/v1/availableAuthProviders',
    loginAuthProviders: '/v1/login/authproviders',
    authProviders: '/v1/authProviders',
    authStatus: '/v1/auth/status',
    tokenRefresh: '/sso/session/tokenrefresh',
};

export const metadata = 'v1/metadata';

export const network = {
    networkBaseline: '/v1/networkbaseline/*', // deployment id
    networkBaselineLock: '/v1/networkbaseline/*/lock',
    networkBaselineUnlock: '/v1/networkbaseline/*/unlock',
    networkBaselinePeers: '/v1/networkbaseline/*/peers',
    networkPoliciesGraph: '/v1/networkpolicies/cluster/*',
    networkGraph: '/v1/networkgraph/cluster/*',
    epoch: '/v1/networkpolicies/graph/epoch',
    generate: '/v1/networkpolicies/generate/*',
    simulate: '/v1/networkpolicies/simulate/*',
    deployment: '/v1/deployments/*',
};

export const policies = {
    policies: '/v1/policies',
    policy: '/v1/policies/*',
    dryrun: '/v1/policies/dryrunjob',
    export: '/v1/policies/export',
    import: '/v1/policies/import',
    reassess: '/v1/policies/reassess',
};

export const roles = {
    list: '/v1/roles',
    mypermissions: 'v1/mypermissions',
};

export const permissionSets = {
    list: '/v1/permissionsets',
};

export const accessScopes = {
    list: '/v1/simpleaccessscopes',
};

export const groups = {
    batch: '/v1/groupsbatch',
    list: '/v1/groups',
};

export const userAttributes = {
    list: '/v1/userattributes/*',
};

export const featureFlags = '/v1/featureflags';

export const integrationHealth = {
    imageIntegrations: '/v1/integrationhealth/imageintegrations',
    signatureIntegrations: '/v1/signatureintegrations',
    notifiers: '/v1/integrationhealth/notifiers',
    externalBackups: '/v1/integrationhealth/externalbackups',
    vulnDefinitions: '/v1/integrationhealth/vulndefinitions',
};

export const integrations = {
    imageIntegrations: '/v1/imageintegrations',
    signatureIntegrations: '/v1/signatureintegrations',
    notifiers: '/v1/notifiers',
    externalBackups: '/v1/externalbackups',
    apiTokens: 'v1/apitokens?revoked=false',
    clusterInitBundles: '/v1/cluster-init/init-bundles',
};

export const integration = {
    apiToken: {
        generate: 'v1/apitokens/generate',
        revoke: '/v1/apitokens/revoke/*',
    },
    clusterInitBundle: {
        generate: 'v1/cluster-init/init-bundles',
        revoke: '/v1/cluster-init/init-bundles/revoke',
    },
};

export const riskAcceptance = {
    getImageVulnerabilities: graphql('getImageVulnerabilities'),
    deferVulnerability: graphql('deferVulnerability'),
    markVulnerabilityFalsePositive: graphql('markVulnerabilityFalsePositive'),
};

export const extensions = {
    diagnostics: '/api/extensions/diagnostics',
};

export const permissions = {
    mypermissions: '/v1/mypermissions',
};
