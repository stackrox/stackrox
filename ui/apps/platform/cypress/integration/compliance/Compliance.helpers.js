import { getRouteMatcherMapForGraphQL, interactAndWaitForResponses } from '../../helpers/request';
import { visit } from '../../helpers/visit';

// path

export const basePath = '/main/compliance';

function getEntitiesPath(entitiesKey) {
    return `${basePath}/${entitiesKey}`;
}

const segmentForEntity = {
    clusters: 'cluster',
    controls: 'control',
    deployments: 'deployment',
    namespaces: 'namespace',
    nodes: 'node',
};

function getEntityPagePath(entitiesKey) {
    return `${basePath}/${segmentForEntity[entitiesKey]}`;
}

// opname

const opnamesWithoutStandards = [
    'clustersCount',
    'namespacesCount',
    'nodesCount',
    'deploymentsCount',
    'runStatuses',
    'getAggregatedResultsAcrossEntity_CLUSTER',
    'getAggregatedResultsByEntity_CLUSTER',
    'getAggregatedResultsAcrossEntity_NAMESPACE',
    'getAggregatedResultsAcrossEntity_NODE',
];

export const routeMatcherMapWithoutStandards =
    getRouteMatcherMapForGraphQL(opnamesWithoutStandards);

// TODO are these reliable after hideScanResults feature?
/*
const opnamesOfStandards = [
    'complianceStandards_CIS_Docker_v1_2_0',
    'complianceStandards_CIS_Kubernetes_v1_5',
    'complianceStandards_HIPAA_164',
    'complianceStandards_NIST_800_190',
    'complianceStandards_NIST_SP_800_53_Rev_4',
    'complianceStandards_PCI_DSS_3_2',
];
*/

const routeMatcherMapForComplianceDashboard = getRouteMatcherMapForGraphQL(opnamesWithoutStandards);

const opnameForEntities = {
    clusters: 'clustersList', // just clusters would be even better, and so on
    controls: 'controls',
    deployments: 'deploymentsList',
    namespaces: 'namespaceList', // singular: too bad, so sad
    nodes: 'nodesList',
};

const opnameForEntity = {
    clusters: 'getCluster',
    deployments: 'getDeployment',
    namespaces: 'getNamespace',
    nodes: 'getNode',
};

// heading

export const headingPlural = {
    clusters: 'Clusters',
    deployments: 'Deployments',
    namespaces: 'Namespaces',
    nodes: 'Nodes',
};

export const headingSingular = {
    clusters: 'Cluster',
    deployments: 'Deployment',
    namespaces: 'Namespace',
    nodes: 'Node',
};

// assert

// assert instead of interact because query might be cached.
export function assertComplianceEntityPage(entitiesKey) {
    cy.location('pathname').should('contain', getEntityPagePath(entitiesKey)); // contain because pathname has id
    cy.get(`h1 + div:contains("${headingSingular[entitiesKey]}")`);
}

// visit

export function visitComplianceDashboard() {
    visit(basePath, routeMatcherMapForComplianceDashboard);

    cy.get(`h1:contains("Compliance")`);
}

/*
 * Assume location is compliance dashboard.
 */
function scanCompliance() {
    const routeMatcherMapForTriggerScan = getRouteMatcherMapForGraphQL(['triggerScan']);
    const routeMatcherMap = {
        ...routeMatcherMapForTriggerScan,
        ...routeMatcherMapForComplianceDashboard,
    };

    const scanButton = '[data-testid="scan-button"]';

    cy.get(scanButton).should('not.have.attr', 'disabled');

    interactAndWaitForResponses(
        () => {
            cy.get(scanButton).click();
            cy.get(scanButton).should('have.attr', 'disabled');
        },
        routeMatcherMap,
        undefined,
        { timeout: 20000 }
    );

    cy.get(scanButton).should('not.have.attr', 'disabled');
}

/*
 * Call directly in first test (of a test file for another container) which assumes that scan results are available.
 * Although compliance test file bends the guideline that tests should not depend on side effects within the same test file,
 * configmanagement test files break the guideline if they assume that compliance test file has already run.
 * Cypress 10 apparently no longer runs tests files in a determinate order.
 */
export function triggerScan() {
    visitComplianceDashboard();
    scanCompliance();
}

/*
 * For example, visitComplianceEntities('clusters')
 */
export function visitComplianceEntities(entitiesKey) {
    const routeMatcherMap = getRouteMatcherMapForGraphQL([
        'searchOptions',
        opnameForEntities[entitiesKey],
    ]);

    visit(getEntitiesPath(entitiesKey), routeMatcherMap);

    cy.get(`h1:contains("${headingPlural[entitiesKey]}")`);
}

/*
 * For example, visitComplianceStandard('CIS Docker v1.2.0')
 */
export function visitComplianceStandard(standardName) {
    const routeMatcherMap = getRouteMatcherMapForGraphQL([
        'searchOptions',
        'getComplianceStandards',
        'controls',
    ]);

    visit(`${getEntitiesPath('controls')}?s[standard]=${standardName}`, routeMatcherMap);

    cy.get(`h1:contains("${standardName}")`);
}

export function interactAndWaitForComplianceEntities(interactionCallback, entitiesKey) {
    const opname = opnameForEntities[entitiesKey];
    const routeMatcherMap = getRouteMatcherMapForGraphQL([opname]);
    interactAndWaitForResponses(interactionCallback, routeMatcherMap);

    cy.location('pathname').should('eq', getEntitiesPath(entitiesKey));
    cy.get(`h1:contains("${headingPlural[entitiesKey]}")`);
}

export function interactAndWaitForComplianceEntityInSidePanel(interactionCallback, entitiesKey) {
    const opname = opnameForEntity[entitiesKey];
    const routeMatcherMap = getRouteMatcherMapForGraphQL([opname]);
    interactAndWaitForResponses(interactionCallback, routeMatcherMap);

    cy.location('pathname').should('contain', getEntitiesPath(entitiesKey)); // contain because pathname has id
}

export function interactAndWaitForComplianceStandard(interactionCallback) {
    const entitiesKey = 'controls';
    const opname = opnameForEntities[entitiesKey];
    const routeMatcherMap = getRouteMatcherMapForGraphQL([opname]);
    interactAndWaitForResponses(interactionCallback, routeMatcherMap);

    cy.location('pathname').should('eq', getEntitiesPath(entitiesKey));
    cy.get('h1 + div:contains("Standard")');
}

// verify

/*
 * For example, verifyDashboardEntityLink('clusters', /^\d+ clusters?/))
 */
export function verifyDashboardEntityLink(entitiesKey, entityRegExp) {
    cy.get('[data-testid="page-header"]')
        .contains('a', entityRegExp)
        .then(($a) => {
            const [, count] = /^(\d+) /.exec($a.text());
            interactAndWaitForComplianceEntities(() => {
                cy.wrap($a).click();
            }, entitiesKey);
            cy.get(`[data-testid="panel-header"]:contains("${count}")`);
        });
}
