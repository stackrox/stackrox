import * as api from '../../constants/apiEndpoints';
import { headingPlural, selectors, url } from '../../constants/VulnManagementPage';
import { hasFeatureFlag } from '../features';

import { visitFromLeftNavExpandable } from '../nav';
import { interactAndWaitForResponses } from '../request';
import { visit } from '../visit';

let opnamesForDashboard = [
    'policiesCount',
    'cvesCount',
    'getNodes',
    'getImages',
    'topRiskyDeployments',
    'topRiskiestImagesOld',
    'topRiskiestImageVulns',
    'frequentlyViolatedPolicies',
    'recentlyDetectedVulnerabilities',
    'recentlyDetectedImageVulnerabilities',
    'mostCommonVulnerabilities',
    'mostCommonImageVulnerabilities',
    'deploymentsWithMostSeverePolicyViolations',
    'clustersWithMostOrchestratorIstioVulnerabilities',
    'clustersWithMostClusterVulnerabilities',
];

if (hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
    opnamesForDashboard = opnamesForDashboard.filter(
        (opname) =>
            opname !== 'clustersWithMostOrchestratorIstioVulnerabilities' &&
            opname !== 'recentlyDetectedVulnerabilities' &&
            opname !== 'topRiskiestImagesOld' &&
            opname !== 'mostCommonVulnerabilities'
    );
} else {
    opnamesForDashboard = opnamesForDashboard.filter(
        (opname) =>
            opname !== 'clustersWithMostClusterVulnerabilities' &&
            opname !== 'recentlyDetectedImageVulnerabilities' &&
            opname !== 'topRiskiestImageVulns' &&
            opname !== 'mostCommonImageVulnerabilities'
    );
}

/*
 * For example, given ['searchOptions', 'getDeployments'] return:
 * {
 *     searchOptions: '/api/graphql?opname=searchOptions',
 *     getDeployments: '/api/graphql?opname=getDeployments',
 * }
 */
function routeMatcherMapForOpnames(opnames) {
    const routeMatcherMap = {};

    opnames.forEach((opname) => {
        routeMatcherMap[opname] = api.graphql(opname);
    });

    return routeMatcherMap;
}

const requestConfigForDashboard = {
    routeMatcherMap: routeMatcherMapForOpnames(opnamesForDashboard),
};

export function visitVulnerabilityManagementDashboardFromLeftNav() {
    visitFromLeftNavExpandable('Vulnerability Management', 'Dashboard', requestConfigForDashboard);

    cy.get('h1:contains("Vulnerability Management")');
}

export function visitVulnerabilityManagementDashboard() {
    visit(url.dashboard, requestConfigForDashboard);

    cy.get('h1:contains("Vulnerability Management")');
}

/*
 * The following keys correspond to url list object in VulnManagementPage.js file.
 */

const opnameForEntity = {
    clusters: 'getCluster',
    components: 'getComponent',
    'image-components': 'getImageComponent',
    'node-components': 'getComponent',
    cves: 'getCve',
    'image-cves': 'getCve',
    'node-cves': 'getCve',
    'cluster-cves': 'getCve',
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

const opnamePrefixForPrimaryAndSecondaryEntities = {
    clusters: 'getCluster_',
    components: 'getComponentSubEntity',
    'image-components': 'getComponentSubEntity',
    'node-components': 'getNodeComponentSubEntity',
    cves: 'getCve',
    'image-cves': 'getCve',
    'node-cves': 'getCve',
    'cluster-cves': 'getCve',
    deployments: 'getDeployment',
    images: 'getImage',
    namespaces: 'getNamespace',
    nodes: 'getNode',
    policies: 'getPolicy',
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
    return `${opnamePrefixForPrimaryAndSecondaryEntities[entitiesKey1]}${typeOfEntity[entitiesKey2]}`;
}

/*
 * For example, visitVulnerabilityManagementEntities('cves')
 * For example, visitVulnerabilityManagementEntities('policies', '?s[Policy]=Fixable Severity at least Important')
 */
export function visitVulnerabilityManagementEntities(entitiesKey, search = '') {
    const requestConfig = {
        routeMatcherMap: routeMatcherMapForOpnames([
            'searchOptions',
            opnameForEntities[entitiesKey],
        ]),
    };

    visit(`${url.list[entitiesKey]}${search}`, requestConfig);

    cy.get(`h1:contains("${headingPlural[entitiesKey]}")`);
}

export function interactAndWaitForVulnerabilityManagementEntities(
    interactionCallback,
    entitiesKey,
    staticResponseForEntities
) {
    const opname = opnameForEntities[entitiesKey];
    const requestConfig = {
        routeMatcherMap: routeMatcherMapForOpnames(['searchOptions', opname]),
    };
    const staticResponseMap = staticResponseForEntities && { [opname]: staticResponseForEntities };

    interactAndWaitForResponses(interactionCallback, requestConfig, staticResponseMap);

    cy.location('pathname').should('eq', url.list[entitiesKey]);
    cy.get(`h1:contains("${headingPlural[entitiesKey]}")`);
}

export function visitVulnerabilityManagementEntityInSidePanel(
    entitiesKey,
    entityId,
    staticResponseForEntity
) {
    const pathname = url.list[entitiesKey];
    const search = `?workflowState[0][t]=${typeOfEntity[entitiesKey]}&workflowState[0][i]=${entityId}`;
    const opname = opnameForEntity[entitiesKey];
    const requestConfig = {
        routeMatcherMap: {
            [opname]: api.graphql(opname),
        },
    };
    const staticResponseMap = staticResponseForEntity && { [opname]: staticResponseForEntity };

    visit(`${pathname}${search}`, requestConfig, staticResponseMap);
}

export function interactAndWaitForVulnerabilityManagementEntity(
    interactionCallback,
    entitiesKey,
    staticResponseForEntity
) {
    const opname = opnameForEntity[entitiesKey];
    const requestConfig = {
        routeMatcherMap: {
            [opname]: api.graphql(opname),
        },
    };
    const staticResponseMap = staticResponseForEntity && { [opname]: staticResponseForEntity };

    interactAndWaitForResponses(interactionCallback, requestConfig, staticResponseMap);
}

export function interactAndWaitForVulnerabilityManagementSecondaryEntities(
    interactionCallback,
    entitiesKey1,
    entitiesKey2,
    staticResponseForSecondaryEntities
) {
    const opname = opnameForPrimaryAndSecondaryEntities(entitiesKey1, entitiesKey2);
    const requestConfig = {
        routeMatcherMap: {
            [opname]: api.graphql(opname),
        },
    };
    const staticResponseMap = staticResponseForSecondaryEntities && {
        [opname]: staticResponseForSecondaryEntities,
    };

    interactAndWaitForResponses(interactionCallback, requestConfig, staticResponseMap);
}

/*
 * resultsFromRegExp: /^(\d+) (\D+)$/.exec(linkText)
 * which assumes that linkText matches a more specific RegExp
 * for example, /^\d+ deployments?$/
 */

function getCountAndNounFromSecondaryEntitiesLinkResults(resultsFromRegExp) {
    return {
        panelHeaderText: resultsFromRegExp[0],
        relatedEntitiesCount: resultsFromRegExp[1],
        relatedEntitiesNoun: resultsFromRegExp[2].toUpperCase(),
    };
}

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
    columnIndex,
    entitiesRegExp2,
    getCountAndNounFromLinkResults = getCountAndNounFromSecondaryEntitiesLinkResults
) {
    // 1. Visit list page for primary entities.
    visitVulnerabilityManagementEntities(entitiesKey1);

    // Find the first link for secondary entities.
    // Plus 1 because of invisible .rt-td.hidden cell.
    cy.get(`.rt-tbody .rt-td:nth-child(${columnIndex + 1})`)
        .contains('a', entitiesRegExp2)
        .then(($a) => {
            const { panelHeaderText, relatedEntitiesCount, relatedEntitiesNoun } =
                getCountAndNounFromLinkResults(/^(\d+) (\D+)$/.exec($a.text()));

            // 2. Visit secondary entities side panel.
            interactAndWaitForResponses(
                () => {
                    cy.wrap($a).click();
                },
                {
                    routeMatcherMap: routeMatcherMapForOpnames([
                        opnameForPrimaryAndSecondaryEntities(entitiesKey1, entitiesKey2),
                    ]),
                }
            );

            cy.get(`${selectors.entityRowHeader}:contains(${panelHeaderText})`);

            // 3. Visit primary entity side panel.
            interactAndWaitForResponses(
                () => {
                    cy.get(selectors.parentEntityInfoHeader).click();
                },
                {
                    // prettier-ignore
                    routeMatcherMap: routeMatcherMapForOpnames([
                        opnameForEntity[entitiesKey1]
                    ]),
                }
            );

            // Tilde because link might be under either Contains or Matches.
            // Match data-testid attribute of link to distinguish 1 IMAGE from 114 IMAGE COMPONENTS.
            const relatedEntitiesSelector = `h2:contains("Related entities") ~ div ul li a[data-testid="${typeOfEntity[entitiesKey2]}-tile-link"]:has('[data-testid="tileLinkSuperText"]:contains("${relatedEntitiesCount}")'):has('[data-testid="tile-link-value"]:contains("${relatedEntitiesNoun}")')`;
            cy.get(relatedEntitiesSelector);

            // 4. Visit single page for primary entity.
            cy.get(selectors.sidePanelExpandButton).click(); // does not make requests

            // 5. Visit list page for secondary entities.
            cy.get(relatedEntitiesSelector).click(); // might make some requests

            cy.get(`${selectors.tabHeader}:contains("${panelHeaderText}")`);
        });
}

/*
 * For filtered secondary entities link, verify panelHeader text only,
 * because related entities has total unfiltered count.
 *
 * For example,
 * 1 Fixable corresponds to any of the following: 1 Image CVE or 1 Node CVE or 1 Platform CVE
 * 2 failing deployments corresponds to 2 deployments
 */
export function verifyFilteredSecondaryEntitiesLink(
    entitiesKey1,
    _entitiesKey2, // unused because response might have been cached
    columnIndex,
    filteredEntitiesRegExp,
    getCountAndNounFromLinkResults
) {
    // 1. Visit list page for primary entities.
    visitVulnerabilityManagementEntities(entitiesKey1);

    // Find the first link for secondary entities.
    cy.get(`.rt-tbody .rt-td:nth-child(${columnIndex + 1})`)
        .contains('a', filteredEntitiesRegExp)
        .then(($a) => {
            const { panelHeaderText } = getCountAndNounFromLinkResults(
                /^(\d+) (\D+)$/.exec($a.text())
            );

            // 2. Visit secondary entities side panel.
            cy.wrap($a).click();

            cy.get(`${selectors.entityRowHeader}:contains(${panelHeaderText})`);
        });
}

/*
 * For fixable CVEs link when primary entities are images,
 * also verify special case that image side panel has risk acceptance tabs.
 *
 * Keep arguments consistent with other functions,
 * expecially in case risk acceptance ever applies to node or platform CVEs.
 */
export function verifyFixableCVEsLinkAndRiskAcceptanceTabs(
    entitiesKey1,
    _entitiesKey2, // unused because response might have been cached
    columnIndex,
    fixableCVEsRegExp,
    getCountAndNounFromLinkResults
) {
    // 1. Visit list page for primary entities.
    visitVulnerabilityManagementEntities(entitiesKey1);

    // Find the first link for secondary entities.
    cy.get(`.rt-tbody .rt-td:nth-child(${columnIndex + 1})`)
        .contains('a', fixableCVEsRegExp)
        .then(($a) => {
            const { panelHeaderText } = getCountAndNounFromLinkResults(
                /^(\d+) (\D+)$/.exec($a.text())
            );

            // 2. Visit secondary entities side panel.
            cy.wrap($a).click();

            cy.get(`${selectors.entityRowHeader}:contains(${panelHeaderText})`);

            // 3. Visit primary entity side panel.
            cy.get(selectors.parentEntityInfoHeader).click();

            // Verify risk acceptance tabs under Image Findings.
            cy.get('.pf-c-tabs .pf-c-tabs__item:eq(0):contains("Observed CVEs")').click({
                force: true,
                waitForAnimations: false,
            });
            cy.get('.pf-c-tabs .pf-c-tabs__item:eq(1):contains("Deferred CVEs")').click({
                force: true,
                waitForAnimations: false,
            });
            cy.get('.pf-c-tabs .pf-c-tabs__item:eq(2):contains("False positive CVEs")').click({
                force: true,
                waitForAnimations: false,
            });
        });
}
