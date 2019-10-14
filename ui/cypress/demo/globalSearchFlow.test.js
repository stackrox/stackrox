import searchSelectors from '../constants/SearchPage';
import { url as policiesURL } from '../constants/PoliciesPage';
import { url as riskURL } from '../constants/RiskPage';
import selectors from '../selectors/index';
import withAuth from '../helpers/basicAuth';

const CVE = 'CVE-2017-5638';

describe('Global Search Flow', () => {
    withAuth();

    it('should search by CVE and get Deployments and Images as results ', () => {
        cy.visit('/');
        cy.get('button', { timeout: 7000 })
            .contains('Search')
            .click();
        cy.get(searchSelectors.searchInput).clear();
        cy.get(searchSelectors.searchInput).type('CVE:{enter}');
        cy.get(searchSelectors.searchInput).type(`${CVE}{enter}`);
        cy.get(searchSelectors.categoryTabs)
            .contains('Deployments')
            .invoke('text')
            .then(text => {
                expect(text).not.to.equal('Deployments (0)');
            });
        cy.get(searchSelectors.categoryTabs)
            .contains('Images')
            .invoke('text')
            .then(text => {
                expect(text).not.to.equal('Images (0)');
            });
    });

    it('should go to the Risk page when the Risk tile is clicked', () => {
        cy.visit('/');
        cy.get('button', { timeout: 7000 })
            .contains('Search')
            .click();
        cy.get(searchSelectors.searchInput).clear();
        cy.get(searchSelectors.searchInput).type('CVE:{enter}');
        cy.get(searchSelectors.searchInput).type(`${CVE}{enter}`);
        cy.get(`${selectors.table.rows}:contains("asset-cache")`)
            .eq(0)
            .find('button')
            .contains('RISK')
            .click();
        cy.url().should('include', riskURL);
        cy.get(selectors.table.rows)
            .eq(0)
            .contains('asset-cache');
    });

    it('Global Search returns results for Policy:CVSS (violations and policies)', () => {
        cy.visit('/');
        cy.get('button', { timeout: 7000 })
            .contains('Search')
            .click();
        cy.get(searchSelectors.searchInput).clear();
        cy.get(searchSelectors.searchInput).type('Policy:{enter}');
        cy.get(searchSelectors.searchInput).type('CVSS{enter}');
        cy.get(searchSelectors.categoryTabs)
            .contains('Violations')
            .invoke('text')
            .then(text => {
                expect(text).not.to.equal('Violations (0)');
            });
        cy.get(searchSelectors.categoryTabs)
            .contains('Policies')
            .invoke('text')
            .then(text => {
                expect(text).not.to.equal('Policies (0)');
            });
    });

    it('From Global Search, user is taken to policies page with CVSS >= 7 selected', () => {
        cy.visit('/');
        cy.get('button', { timeout: 7000 })
            .contains('Search')
            .click();
        cy.get(searchSelectors.searchInput).clear();
        cy.get(searchSelectors.searchInput).type('Policy:{enter}');
        cy.get(searchSelectors.searchInput).type('CVSS{enter}');
        cy.get(`${selectors.table.rows}:contains("Fixable CVSS >= 7")`)
            .eq(0)
            .find('button')
            .contains('POLICIES')
            .click({ force: true });
        cy.url().should('include', policiesURL);
        cy.get(selectors.table.activeRow).contains('Fixable CVSS >= 7');
        cy.get(selectors.search.chips)
            .eq(0)
            .contains('Policy:');
        cy.get(selectors.search.chips)
            .eq(1)
            .contains('CVSS');
        cy.get(selectors.table.rows).then(rows => {
            expect(rows).to.have.length(2);
        });
    });
});
