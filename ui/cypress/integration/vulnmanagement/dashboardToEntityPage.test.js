import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';

function validatePresenceOfTabsAndLinks(selector, relatedEntities) {
    cy.get(selector).each(($el, i) => {
        if (relatedEntities[i] === 'Policies' || relatedEntities[i] === 'POLICIES') {
            expect($el.text()).contains(
                relatedEntities[i] || 'policy' || 'POLICY',
                'expected text is displayed'
            );
        } else {
            expect($el.text()).contains(relatedEntities[i], 'expected text is displayed');
        }
    });
}

function validateRelatedEntitiesValuesWithTabsHeaders(entityName) {
    cy.get(selectors.getTileLink(entityName))
        .invoke('text')
        .then(value => {
            cy.get(selectors.getAllClickableTileLinks(entityName)).click();
            cy.get(selectors.tabHeader)
                .invoke('text')
                .then(text => {
                    expect(parseInt(text, 10)).to.equals(
                        parseInt(value, 10),
                        `number of ${entityName}(s) in the list matches the overview tab tile link`
                    );
                });
        });
}

function validateWithAParentSelector(
    entityName,
    urlToVerify,
    topRowSelector,
    rowIndex,
    tabLinksList,
    tileLinksList
) {
    cy.visit(url.dashboard);
    cy.get(topRowSelector)
        .eq(rowIndex)
        .invoke('text')
        .then(value => {
            // trim the first 3 chars from the front of the text, to get the image name to compare in the detail view
            //   e.g., "1. k8s.gcr.io/coredns:1.3.1" trimmed becomes "k8s.gcr.io/coredns:1.3.1"
            let rowText;
            if (entityName === 'image') rowText = value.slice(2, value.length - 1).trimLeft();
            if (entityName === 'cve')
                rowText = value
                    .slice(2, value.length - 1)
                    .split('/')[0]
                    .trimLeft()
                    .trimRight();
            cy.get(topRowSelector)
                .eq(rowIndex)
                .click();
            cy.url().should('contain', urlToVerify);
            cy.get('[data-test-id="header-text"]').should('have.text', rowText);
            validatePresenceOfTabsAndLinks(selectors.tabLinks, tabLinksList);
            validatePresenceOfTabsAndLinks(selectors.allTileLinks, tileLinksList);
            for (let i = 0; i < tileLinksList.length; i += 1) {
                cy.get(selectors.tabLinks)
                    .find(`:contains('Overview')`)
                    .click();
                validateRelatedEntitiesValuesWithTabsHeaders(tileLinksList[i]);
            }
        });
}

function validateWithActualSelector(
    entityName,
    urlToVerify,
    topRowSelector,
    tabLinksList,
    tileLinksList
) {
    cy.visit(url.dashboard);
    cy.get(topRowSelector)
        .invoke('text')
        .then(value => {
            // trim the first 3 chars from the front of the text, to get the image name to compare in the detail view
            //   e.g., "1. k8s.gcr.io/coredns:1.3.1" trimmed becomes "k8s.gcr.io/coredns:1.3.1"
            let rowText;
            if (entityName === 'image') rowText = value.slice(2, value.length - 1).trimLeft();
            if (entityName === 'cve')
                rowText = value
                    .slice(2, value.length - 1)
                    .split('/')[0]
                    .trimLeft()
                    .trimRight();
            cy.get(topRowSelector).click();
            cy.url().should('contain', urlToVerify);
            cy.get('[data-test-id="header-text"]').should('have.text', rowText);
            validatePresenceOfTabsAndLinks(selectors.tabLinks, tabLinksList);
            validatePresenceOfTabsAndLinks(selectors.allTileLinks, tileLinksList);
            for (let i = 0; i < tileLinksList.length; i += 1) {
                cy.get(selectors.tabLinks)
                    .find(`:contains('Overview')`)
                    .click();
                validateRelatedEntitiesValuesWithTabsHeaders(tileLinksList[i]);
            }
        });
}

describe('Vuln Management Dashboard Page To Entity Page Navigation Validation', () => {
    withAuth();
    if (checkFeatureFlag('ROX_VULN_MGMT_UI', true)) {
        it('validate data consistency for top riskiest images widget data row onwards', () => {
            cy.visit(url.dashboard);
            cy.get(selectors.getWidget('Top Riskiest Images'))
                .find(selectors.viewAllButton)
                .click();
            cy.get(selectors.numCVEColLink)
                .eq(2)
                .invoke('text')
                .then(value => {
                    if (value === 'No CVEs') {
                        validateWithAParentSelector(
                            'image',
                            url.list.image,
                            selectors.dataRowLink,
                            0,
                            [
                                'Overview',
                                'deployments',
                                'components',
                                'CVES',
                                'Fixable CVEs',
                                'Dockerfile'
                            ],
                            ['DEPLOYMENT', 'COMPONENT']
                        );
                    } else {
                        validateWithAParentSelector(
                            'image',
                            url.list.image,
                            selectors.dataRowLink,
                            0,
                            [
                                'Overview',
                                'deployments',
                                'components',
                                'CVES',
                                'Fixable CVEs',
                                'Dockerfile'
                            ],
                            ['DEPLOYMENT', 'COMPONENT', 'CVE']
                        );
                    }
                });
        });

        it('validate data consistency for most common vulnerabilities widget data row onwards', () => {
            validateWithActualSelector(
                'cve',
                url.list.cve,
                selectors.topMostRowMCV,
                ['Overview', 'components', 'images', 'deployments'],
                ['COMPONENT', 'IMAGE', 'DEPLOYMENT']
            );
        });

        it('validate data consistency for recently detected vulnerabilities widget data row onwards', () => {
            validateWithActualSelector(
                'cve',
                url.list.cve,
                selectors.topMostRowRDV,
                ['Overview', 'components', 'images', 'deployments'],
                ['COMPONENT', 'IMAGE', 'DEPLOYMENT']
            );
        });

        it('validate data consistency for frequently violated policies widget data row onwards', () => {
            validateWithActualSelector(
                'cve',
                url.list.policy,
                selectors.topMostRowFVP,
                ['Overview', 'deployments'],
                ['DEPLOYMENT']
            );
        });

        it('validate data consistency for deployments with most severe policy violations widget data row onwards', () => {
            validateWithActualSelector(
                'cve',
                url.list.deployment,
                selectors.topMostRowMSPV,

                [
                    'Overview',
                    'images',
                    'components',
                    'failing policies',
                    'CVES',
                    'Policies',
                    'Fixable CVEs'
                ],
                ['POLICIES', 'IMAGE', 'COMPONENT', 'CVE']
            );
        });
    }
});
