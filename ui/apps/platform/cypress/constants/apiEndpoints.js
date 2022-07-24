import upperFirst from 'lodash/upperFirst';

export const graphql = (operationName) => `/api/graphql?opname=${operationName}`;

// TODO Encapsulate exceptions in test constants for Configuration Management.

const operationNamePlural = {
    deployments: 'getDeployments',
    roles: 'k8sRoles',
};

const operationNameSingular = {
    control: 'controlById',
    role: 'k8sRole',
};

/*
 * graphqlPluralEntity('serviceAccounts') === 'api/graphql?opname=serviceAccounts'
 */
export function graphqlPluralEntity(entityPlural) {
    const operationName = operationNamePlural[entityPlural] ?? entityPlural;
    return graphql(operationName);
}

/*
 * graphqlSingularEntity('serviceAccount') === 'api/graphql?opname=getServiceAccount'
 *
 * Note that lodash capitalize converts the remaining characters to lower case.
 */
export function graphqlSingularEntity(entitySingular) {
    const operationName =
        operationNameSingular[entitySingular] ?? `get${upperFirst(entitySingular)}`;
    return graphql(operationName);
}

// TODO graphqlSubentity(entity, subentity)

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
    results: '/v1/search?query=*',
    options: '/v1/search/metadata/options*',
    autocomplete: 'v1/search/autocomplete*',
    autocompleteBySearch: (searchObj, category) =>
        `v1/search/autocomplete?query=${searchObjToQuery(searchObj)}&categories=${category}`,
    graphqlOps: {
        autocomplete: 'autocomplete',
    },
};

export const alerts = {
    countsByCluster: '/v1/alerts/summary/counts?*group_by=CLUSTER*',
    countsByCategory: '/v1/alerts/summary/counts?*group_by=CATEGORY*',
    alerts: '/v1/alerts',
    alertsWithQuery: '/v1/alerts?query=*',
    alertById: '/v1/alerts/*',
    resolveAlert: '/v1/alerts/*/resolve',
    alertsCountWithQuery: '/v1/alertscount?query=*',
    pageSearchAutocomplete: (searchObj) => search.autocompleteBySearch(searchObj, 'ALERTS'),
    graphqlOps: {
        addTags: 'addAlertTags',
        getTags: 'getAlertTags',
        tagsAutocomplete: 'autocomplete',
        bulkAddAlertTags: 'bulkAddAlertTags',
        removeTags: 'removeAlertTags',
    },
};

export const clusters = {
    single: 'v1/clusters/**',
    list: 'v1/clusters',
    clusterDefaults: '/v1/cluster-defaults',
    sensorUpgradesConfig: '/v1/sensorupgrades/config',
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
    loginAuthProviders: '/v1/login/authproviders',
    authProviders: '/v1/authProviders*',
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
    summaryCounts: graphql('summary_counts'),
};

export const metadata = 'v1/metadata';

export const network = {
    networkBaselineLock: '/v1/networkbaseline/*/lock',
    networkBaselineUnlock: '/v1/networkbaseline/*/unlock',
    networkBaselinePeers: '/v1/networkbaseline/*/peers',
    networkBaselineStatus: '/v1/networkbaseline/*/status',
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

const complianceEntitiesOp = {
    clusters: 'clustersList',
    deployments: 'deploymentsList',
    namespaces: 'namespaceList', // singular: too bad, so sad
    nodes: 'nodesList',
};

export const compliance = {
    // For example, graphqlEntities('clusters')
    graphqlEntities: (key) => graphql(complianceEntitiesOp[key]),
    export: {
        csv: '/api/compliance/export/csv',
    },
};

export const logs = '/api/logimbue';

export const featureFlags = '/v1/featureflags';

export const configMgmt = {
    // Use graphqlPluralEntity or graphqlSingularEntity.
};

/*
 * The following keys correspond to url list object in VulnManagementPage.js file.
 */

const vulnMgmtEntityOp = {
    clusters: 'getCluster',
    components: 'getComponent',
    cves: 'getCve',
    deployments: 'getDeployment',
    images: 'getImage',
    namespaces: 'getNamespace',
    nodes: 'getNode',
    policies: 'getPolicy',
};

const vulnMgmtEntitiesOp = {
    clusters: 'getClusters',
    components: 'getComponents',
    'image-components': 'getImageComponents',
    'node-components': 'getNodeComponents',
    cves: 'getCves',
    'image-cves': 'getImageCves',
    'node-cves': 'getNodeCves',
    'cluster-cves': 'getClusterCves',
    deployments: 'getDeployments',
    images: 'getImages',
    namespaces: 'getNamespaces',
    nodes: 'getNodes',
    policies: 'getPolicies',
};

const vulnMgmtEntitiesPrefix = {
    clusters: 'getCluster_',
    components: 'getComponentSubEntity',
    cves: 'getCve',
    deployments: 'getDeployment',
    images: 'getImage',
    namespaces: 'getNamespace',
    nodes: 'getNode',
    policies: 'getPolicy',
};

export const vulnMgmt = {
    // For example, graphqlEntity('clusters')
    graphqlEntity: (key) => graphql(vulnMgmtEntityOp[key]),
    // For example, graphqlEntities('clusters')
    graphqlEntities: (key) => graphql(vulnMgmtEntitiesOp[key]),
    // For example, graphqlEntities2('clusters', 'CVE')
    // prettier-ignore
    graphqlEntities2: (key1, key2) => graphql(`${vulnMgmtEntitiesPrefix[key1]}${key2}`),
    graphqlOps: {
        getCves: 'getCves',
        getPolicies: 'getPolicies',
        getPolicy: 'getPolicy',
        getImage: 'getImage',
        getNode: 'getNode',
        getDeploymentCOMPONENT: 'getDeploymentCOMPONENT',
        getFixableCvesForEntity: 'getFixableCvesForEntity',
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

export const report = {
    configurations: '/v1/report/configurations*',
    configurationsCount: '/v1/report-configurations-count*',
};

export const riskAcceptance = {
    getImageVulnerabilities: graphql('getImageVulnerabilities'),
    deferVulnerability: graphql('deferVulnerability'),
    markVulnerabilityFalsePositive: graphql('markVulnerabilityFalsePositive'),
};

export const system = {
    config: '/v1/config',
    configPublic: '/v1/config/public',
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
