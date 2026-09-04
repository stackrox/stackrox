import { selectors } from './VulnerabilityManagement.selectors';
import withAuth from '../../helpers/basicAuth';
import {
    expectRequestedSort,
    getRouteMatcherMapForGraphQL,
    interceptAndWatchRequests,
} from '../../helpers/request';
import { visit } from '../../helpers/visit';
import { hasTableColumnHeadings } from './VulnerabilityManagement.helpers';

const nodeCvesFixturePath = 'vulnmanagement/nodeCves.json';

function visitNodeCvesWithMockedData() {
    const routeMatcherMap = getRouteMatcherMapForGraphQL(['searchOptions', 'getNodeCves']);
    const staticResponseMap = {
        getNodeCves: { fixture: nodeCvesFixturePath },
    };
    visit('/main/vulnerability-management/node-cves', routeMatcherMap, staticResponseMap);
    cy.get('h1:contains("Node CVEs")');
}

describe('Vulnerability Management Node CVEs', () => {
    withAuth();

    it('should display table columns', () => {
        visitNodeCvesWithMockedData();

        hasTableColumnHeadings([
            '', // checkbox
            '', // hidden
            'CVE',
            'Operating System',
            'Fixable',
            'Severity',
            'CVSS Score',
            'Env. Impact',
            'Impact Score',
            'Entities',
            'Discovered Time',
            'Published',
            '', // hidden
        ]);
    });

    it('should sort the CVSS Score column', () => {
        visitNodeCvesWithMockedData();

        const thSelector = '.rt-th:contains("CVSS Score")';

        cy.get(thSelector).should('have.class', '-sort-desc');

        const routeMatcherMap = getRouteMatcherMapForGraphQL(['getNodeCves']);
        const staticResponseMap = {
            getNodeCves: { fixture: nodeCvesFixturePath },
        };
        interceptAndWatchRequests(routeMatcherMap, staticResponseMap).then(
            ({ waitAndYieldRequestBodyVariables }) => {
                cy.get(thSelector).click();
                cy.location('search').should('eq', '?sort[0][id]=CVSS&sort[0][desc]=false');
                cy.get(thSelector).should('have.class', '-sort-asc');

                waitAndYieldRequestBodyVariables().then(
                    expectRequestedSort({ field: 'CVSS', reversed: false })
                );
            }
        );

        cy.get(thSelector).click();
        cy.location('search').should('eq', '?sort[0][id]=CVSS&sort[0][desc]=true');
        cy.get(thSelector).should('have.class', '-sort-desc');
    });

    it('should display vulnerability descriptions', () => {
        visitNodeCvesWithMockedData();

        cy.get(selectors.cveDescription).should('exist');
        cy.get(`${selectors.cveDescription}:contains("No description available")`).should(
            'not.exist'
        );
    });

    it('should display links for nodes', () => {
        visitNodeCvesWithMockedData();

        cy.get('.rt-tbody .rt-td')
            .contains('a', /^\d+ nodes?$/)
            .should('exist');
    });

    it('should display links for node-components', () => {
        visitNodeCvesWithMockedData();

        cy.get('.rt-tbody .rt-td')
            .contains('a', /^\d+ node components?$/)
            .should('exist');
    });
});
