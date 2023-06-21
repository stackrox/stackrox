import { selectors } from '../../constants/VulnManagementPage';
import { hasFeatureFlag } from '../features';

import { visitFromLeftNavExpandable } from '../nav';
import { getRouteMatcherMapForGraphQL, interactAndWaitForResponses } from '../request';
import { visit } from '../visit';

const opnamesForDashboard = [
    'cvesCount',
    'getNodes',
    'getImages',
    'topRiskyDeployments',
    'topRiskiestImageVulns',
    'recentlyDetectedImageVulnerabilities',
    'mostCommonImageVulnerabilities',
    'clustersWithMostClusterVulnerabilities',
];

const routeMatcherMapForVulnerabilityManagementDashboard =
    getRouteMatcherMapForGraphQL(opnamesForDashboard);

/*
 * The following keys are path segments which correspond to entityKeys arguments of functions below.
 */

const opnameForEntity = {
    clusters: 'getCluster',
    components: 'getComponent',
    'image-components': 'getImageComponent',
    'node-components': 'getNodeComponent',
    cves: 'getCve',
    'image-cves': 'getImageCve',
    'node-cves': 'getNodeCve',
    'cluster-cves': 'getClusterCve',
    deployments: 'getDeployment',
    images: 'getImage',
    namespaces: 'getNamespace',
    nodes: 'getNode',
    policies: 'getPolicy',
};

const opnameForEntities = {
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

// Headings on entities pages has sentence case for entity type.
const headingPlural = {
    clusters: 'Clusters',
    components: 'Components',
    'image-components': 'Image components',
    'node-components': 'Node components',
    cves: 'CVES',
    'image-cves': 'Image CVES', // TODO uppercase S from pluralize
    'node-cves': 'Node CVES', // TODO uppercase S from pluralize
    'cluster-cves': 'Platform CVES', // TODO uppercase S from pluralize
    deployments: 'Deployments',
    images: 'Images',
    namespaces: 'Namespaces',
    nodes: 'Nodes',
    policies: 'Policies',
};

// For table links and table headings.
const countNounRegExp = {
    clusters: /^\d+ clusters?$/,
    components: /^\d+ components?$/, // TODO delete later for ROX-17764
    'image-components': /^\d+ image components?$/,
    'node-components': /^\d+ node components?$/,
    // For table links, verifyConditionalCVEs uses allCVEsRegExp and fixableCVEsRegExp.
    cves: /^\d+ CVES?$/, // TODO delete later for ROX-17764
    'image-cves': /^\d+ Image CVES?$/, // TODO investigate inconsistent case
    'node-cves': /^\d+ Node CVES?$/, // TODO investigate inconsistent case
    'cluster-cves': /^\d+ Platform CVES?$/, // TODO investigate inconsistent case
    deployments: /^\d+ deployments?$/,
    images: /^\d+ images?$/,
    namespaces: /^\d+ namespaces?$/,
    nodes: /^\d+ nodes?$/,
    // policies TODO delete from sibling objects because obsolete after #6235
};

const typeOfEntity = {
    clusters: 'CLUSTER',
    components: 'COMPONENT',
    'image-components': 'IMAGE_COMPONENT',
    'node-components': 'NODE_COMPONENT',
    cves: 'CVE',
    'image-cves': 'IMAGE_CVE',
    'node-cves': 'NODE_CVE',
    'cluster-cves': 'CLUSTER_CVE',
    deployments: 'DEPLOYMENT',
    images: 'IMAGE',
    namespaces: 'NAMESPACE',
    nodes: 'NODE',
    policies: 'POLICY',
};

/*
 * For example, given 'deployments' and 'image' return: 'getDeploymentIMAGE'
 */
function opnameForPrimaryAndSecondaryEntities(entitiesKey1, entitiesKey2) {
    return `${opnameForEntity[entitiesKey1]}${typeOfEntity[entitiesKey2]}`;
}

const basePath = '/main/vulnerability-management'; // dashboard

function getEntitiesPath(entitiesKey, search = '') {
    return `${basePath}/${entitiesKey}${search}`;
}

function getEntityPath(entitiesKey, entityId) {
    const entityType = typeOfEntity[entitiesKey];
    const search = `?workflowState[0][t]=${entityType}&workflowState[0][i]=${entityId}`;
    return getEntitiesPath(entitiesKey, search);
}

export function visitVulnerabilityManagementDashboardFromLeftNav() {
    const oldVulnMgmtNavText = hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')
        ? 'Vulnerability Management (1.0)'
        : 'Vulnerability Management';
    visitFromLeftNavExpandable(
        oldVulnMgmtNavText,
        'Dashboard',
        routeMatcherMapForVulnerabilityManagementDashboard
    );

    cy.location('pathname').should('eq', basePath);
    cy.location('search').should('eq', '');
    cy.get('h1:contains("Vulnerability Management")');
}

export function visitVulnerabilityManagementDashboard() {
    visit(basePath, routeMatcherMapForVulnerabilityManagementDashboard);

    cy.get('h1:contains("Vulnerability Management")');
}

/*
 * For example, visitVulnerabilityManagementEntities('cves')
 * For example, visitVulnerabilityManagementEntities('policies', '?s[Policy]=Fixable Severity at least Important')
 */
export function visitVulnerabilityManagementEntities(entitiesKey) {
    const routeMatcherMap = getRouteMatcherMapForGraphQL([
        'searchOptions',
        opnameForEntities[entitiesKey],
    ]);

    const interceptions = visit(getEntitiesPath(entitiesKey), routeMatcherMap);

    cy.get(`h1:contains("${headingPlural[entitiesKey]}")`);

    return interceptions;
}

export function visitVulnerabilityManagementEntitiesWithSearch(entitiesKey, search) {
    const routeMatcherMap = getRouteMatcherMapForGraphQL([
        'searchOptions',
        opnameForEntities[entitiesKey],
    ]);

    visit(getEntitiesPath(entitiesKey, search), routeMatcherMap);

    cy.get(`h1:contains("${headingPlural[entitiesKey]}")`);
}

export function interactAndWaitForVulnerabilityManagementEntities(
    interactionCallback,
    entitiesKey,
    staticResponseForEntities
) {
    /*
     * Unlike visit function above, omit searchOptions request
     * to support tests to sort the table by a column.
     * By the way, the tests do not call this function for the click
     * to restore initial sorting, because the response has been cached.
     */
    const opname = opnameForEntities[entitiesKey];
    const routeMatcherMap = getRouteMatcherMapForGraphQL([opname]);
    const staticResponseMap = staticResponseForEntities && { [opname]: staticResponseForEntities };

    interactAndWaitForResponses(interactionCallback, routeMatcherMap, staticResponseMap);

    cy.location('pathname').should('eq', getEntitiesPath(entitiesKey));
    cy.get(`h1:contains("${headingPlural[entitiesKey]}")`);
}

export function visitVulnerabilityManagementEntityInSidePanel(
    entitiesKey,
    entityId,
    staticResponseForEntity
) {
    const opname = opnameForEntity[entitiesKey];
    const routeMatcherMap = getRouteMatcherMapForGraphQL([opname]);
    const staticResponseMap = staticResponseForEntity && { [opname]: staticResponseForEntity };

    visit(getEntityPath(entitiesKey, entityId), routeMatcherMap, staticResponseMap);
}

export function interactAndWaitForVulnerabilityManagementEntity(
    interactionCallback,
    entitiesKey,
    staticResponseForEntity
) {
    const opname = opnameForEntity[entitiesKey];
    const routeMatcherMap = getRouteMatcherMapForGraphQL([opname]);
    const staticResponseMap = staticResponseForEntity && { [opname]: staticResponseForEntity };

    interactAndWaitForResponses(interactionCallback, routeMatcherMap, staticResponseMap);
}

export function interactAndWaitForVulnerabilityManagementSecondaryEntities(
    interactionCallback,
    entitiesKey1,
    entitiesKey2,
    staticResponseForSecondaryEntities
) {
    const opname = opnameForPrimaryAndSecondaryEntities(entitiesKey1, entitiesKey2);
    const routeMatcherMap = getRouteMatcherMapForGraphQL([opname]);
    const staticResponseMap = staticResponseForSecondaryEntities && {
        [opname]: staticResponseForSecondaryEntities,
    };

    interactAndWaitForResponses(interactionCallback, routeMatcherMap, staticResponseMap);
}

/*
 * resultsFromRegExp: /^(\d+) (\D+)$/.exec(linkText)
 * which assumes that linkText matches a more specific RegExp
 * for example, /^\d+ deployments?$/
 */

// After accessibility-related changes to case of entity types,
// getCountAndNoun functions might become obsolete, as follows:

// 1. Done for ROX-17001

// 2. TODO because visible text is better than data-testid attribute.
//    Replace selector which has data-testid attribute
//    with contains method and RegExp for exact match:
//    correct case entity type noun with optional plural suffix.

export function getCountAndNounFromImageCVEsLinkResults([, count]) {
    return {
        panelHeaderText: `${count} Image ${count === '1' ? 'CVE' : 'CVES'}`,
        relatedEntitiesCount: count,
        relatedEntitiesNoun: count === '1' ? 'IMAGE CVE' : 'IMAGE CVES',
    };
}

export function getCountAndNounFromNodeCVEsLinkResults([, count]) {
    return {
        panelHeaderText: `${count} Node ${count === '1' ? 'CVE' : 'CVES'}`,
        relatedEntitiesCount: count,
        relatedEntitiesNoun: count === '1' ? 'NODE CVE' : 'NODE CVES',
    };
}

/*
 * Keys for primary and secondary entities are plural page address segments.
 * For example, primary 'namespaces' and secondary 'deployments'
 * corresponds to the following pages:
 * /main/vulnerability-management/namespaces
 * /main/vulnerability-management/namespace/id/deployments
 *
 * columnIndex is one-based but would be dataLabel in PatternFly.
 *
 * entitiesRegExp2 matches links in primary entities table to secondary entities.
 * For example, /^\d+ deployments?$/
 *
 * getCountAndNounFromLinkResults optioanl function provides the noun.
 * For example,
 * Noun is not in link text: /^\d+ CVEs?$/ or /^\d+ Fixable$/
 * Noun differs from link text: /^\d+ failing deployments?$/
 */
export function verifySecondaryEntities(
    entitiesKey1,
    entitiesKey2,
    columnIndex // one-based index includes checkbox, hidden, invisible
) {
    // 1. Visit list page for primary entities.
    visitVulnerabilityManagementEntities(entitiesKey1);

    // 2. Find the first link for secondary entities.
    verifyTableLink(entitiesKey1, entitiesKey2, columnIndex);
}

/*
 * Verify panelHeader text, and then visit related entities pages,
 */
function verifyTableLink(
    entitiesKey1,
    entitiesKey2,
    columnIndex, // one-based index includes checkbox, hidden, invisible
    entitiesRegExp2
) {
    // Find the first link for secondary entities.
    cy.get(selectors.getTableDataColumnSelector(columnIndex))
        .contains('a', entitiesRegExp2 ?? countNounRegExp[entitiesKey2])
        .then(($a) => {
            // 2. Visit secondary entities side panel.
            interactAndWaitForResponses(() => {
                cy.wrap($a).click();
            }, getRouteMatcherMapForGraphQL([opnameForPrimaryAndSecondaryEntities(entitiesKey1, entitiesKey2)]));

            cy.get('[data-testid="side-panel"] [data-testid="panel-header"]').contains(
                'div',
                countNounRegExp[entitiesKey2]
            );

            // 3. Visit primary entity side panel.
            interactAndWaitForResponses(() => {
                cy.get(selectors.parentEntityInfoHeader).click();
            }, getRouteMatcherMapForGraphQL([opnameForEntity[entitiesKey1]]));

            // Tilde because link might be under either Contains or Matches.
            // Match data-testid attribute of link to distinguish 1 IMAGE from 114 IMAGE COMPONENTS.
            // Omit has for visible text of count or name of entity because it might have changed (especially for deployments).
            const relatedEntitiesSelector = `h2:contains("Related entities") ~ div ul li a[data-testid="${typeOfEntity[entitiesKey2]}-tile-link"]`;
            cy.get(relatedEntitiesSelector);

            // 4. Visit single page for primary entity.
            cy.get(selectors.sidePanelExternalLinkButton).click(); // does not make requests

            // 5. Visit list page for secondary entities.
            cy.get(relatedEntitiesSelector).click(); // might make some requests

            cy.get(
                `li[data-testid="grouped-tab"] a[data-testid="tab"].active:contains("${headingPlural[entitiesKey2]}")`
            );
            cy.get('[data-testid="panel"]').contains('div', countNounRegExp[entitiesKey2]);
        });
}

const allCVEsRegExp = /^\d+ CVEs?$/;
const fixableCVEsRegExp = /^\d+ Fixable$/;

/*
 * Conditional test of either links for CVEs or text for No CVEs.
 * 1. Prefer link for Fixable CVEs and visit only side panel (shallow).
 * 2. Otherwise link for all CVEs and visit related entities pages (deep).
 * 3. Otherwise text for No CVEs.
 */
export function verifyConditionalCVEs(
    entitiesKey1,
    entitiesKey2,
    columnIndex, // one-based index includes checkbox, hidden, invisible
    vulnCounterKey
) {
    // 1. Visit list page for primary entities.
    // The first interception is ignored because for searchOptions request.
    // The second interception is for entitiesKey1 request.
    visitVulnerabilityManagementEntities(entitiesKey1).then(([, { response }]) => {
        const { results } = response.body.data;

        // Check sources of truth whether or not to assert existence of links.
        const hasFixableCVEs = results.some((result) => result[vulnCounterKey]?.all?.fixable > 0);
        const hasCVEs = results.some((result) => result[vulnCounterKey]?.all?.total > 0);

        if (hasFixableCVEs) {
            // If at least one of entitiesKey1 has fixable CVEs, then CVEs link exists.
            cy.get(selectors.getTableDataColumnSelector(columnIndex))
                .contains('a', allCVEsRegExp)
                .should('exist');

            verifyTableLink(entitiesKey1, entitiesKey2, columnIndex, fixableCVEsRegExp);
        } else if (hasCVEs) {
            // Fixable link does not exist in any row of entityKeys1 list.
            cy.get(selectors.getTableDataColumnSelector(columnIndex))
                .contains('a', fixableCVEsRegExp)
                .should('not.exist');

            verifyTableLink(entitiesKey1, entitiesKey2, columnIndex, allCVEsRegExp);
        } else {
            // Neither link exists in any row of entitiesKey1 list.
            cy.get(selectors.getTableDataColumnSelector(columnIndex))
                .contains('a', fixableCVEsRegExp)
                .should('not.exist');
            cy.get(selectors.getTableDataColumnSelector(columnIndex))
                .contains('a', allCVEsRegExp)
                .should('not.exist');
            cy.get(`${selectors.getTableDataColumnSelector(columnIndex)}:contains("No CVEs")`);
        }
    });
}

// table

/*
 * Assert table column headings in order with empty string if no text (for example, checkbox).
 */
export function hasTableColumnHeadings(tableColumnHeadings) {
    tableColumnHeadings.forEach((tableColumnHeading, index0) => {
        const index1 = index0 + 1; // nth-child selector has one-based index
        if (tableColumnHeading.length === 0) {
            cy.get(`.rt-th:nth-child(${index1})`);
        } else {
            cy.get(`.rt-th:nth-child(${index1}):contains("${tableColumnHeading}")`);
        }
    });

    cy.get('.rt-th').should('have.length', tableColumnHeadings.length);
}
