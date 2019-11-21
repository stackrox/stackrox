import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';

function validatePresenceOfTabsAndLinks(selector, relatedEntities) {
    cy.get(selector).each(($el, i) => {
        expect($el.text()).contains(relatedEntities[i], 'expected text is displayed');
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

describe('Vuln Management Dashboard Page To Entity Page Navigation Validation', () => {
    withAuth();
    it('validate data consistency for top riskiest images widget data row onwards', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Top Riskiest Images'))
            .find(selectors.dataRowLink)
            .eq(0)
            .invoke('text')
            .then(value => {
                // trim the first 3 chars from the front of the text, to get the image name to compare in the detail view
                //   e.g., "1. k8s.gcr.io/coredns:1.3.1" trimmed becomes "k8s.gcr.io/coredns:1.3.1"
                const imageName = value.slice(3, value.length - 1);
                cy.get(selectors.getWidget('Top Riskiest Images'))
                    .find(selectors.dataRowLink)
                    .eq(0)
                    .click();
                cy.url().should('contain', url.list.image);
                cy.get('[data-test-id="header-text"]').should('have.text', imageName);
                validatePresenceOfTabsAndLinks(selectors.tabLinks, [
                    'Overview',
                    'deployments',
                    'components',
                    'CVES',
                    'Fixable CVEs',
                    'Dockerfile'
                ]);
                validatePresenceOfTabsAndLinks(selectors.allTileLinks, [
                    'DEPLOYMENT',
                    'COMPONENT',
                    'CVE'
                ]);
                validateRelatedEntitiesValuesWithTabsHeaders('DEPLOYMENT');
                cy.get(selectors.tabLinks)
                    .find(`:contains('Overview')`)
                    .click();
                validateRelatedEntitiesValuesWithTabsHeaders('CVE');
                cy.get(selectors.tabLinks)
                    .find(`:contains('Overview')`)
                    .click();
                validateRelatedEntitiesValuesWithTabsHeaders('COMPONENT');
            });
    });
});
