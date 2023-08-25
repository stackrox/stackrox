import withAuth from '../../helpers/basicAuth';
import {
    assertSortedItems,
    callbackForPairOfAscendingNumberValuesFromElements,
    callbackForPairOfDescendingNumberValuesFromElements,
} from '../../helpers/sort';
import {
    hasTableColumnHeadings,
    interactAndWaitForVulnerabilityManagementEntities,
    visitVulnerabilityManagementEntities,
    visitVulnerabilityManagementEntityInSidePanel,
} from './VulnerabilityManagement.helpers';
import { selectors } from './VulnerabilityManagement.selectors';

const entitiesKey = 'nodes';

describe('Vulnerability Management Nodes', () => {
    withAuth();

    it('should display table columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasTableColumnHeadings([
            '', // hidden
            'Node',
            'Node CVEs',
            'Top CVSS',
            'Cluster',
            'Operating System',
            'Container Runtime',
            'Join Time',
            'Scan Time',
            'Risk Priority',
        ]);
    });

    it('should sort the Risk Priority column', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        const thSelector = '.rt-th:contains("Risk Priority")';
        const tdSelector = '.rt-td:nth-child(10)';

        // 0. Initial table state indicates that the column is sorted ascending.
        cy.get(thSelector).should('have.class', '-sort-asc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfAscendingNumberValuesFromElements);
        });

        // 1. Sort descending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should(
            'eq',
            '?sort[0][id]=Node%20Risk%20Priority&sort[0][desc]=true'
        );

        cy.get(thSelector).should('have.class', '-sort-desc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingNumberValuesFromElements);
        });

        // 2. Sort ascending by the column.
        cy.get(thSelector).click(); // no request because initial response has been cached
        cy.location('search').should(
            'eq',
            '?sort[0][id]=Node%20Risk%20Priority&sort[0][desc]=false'
        );

        cy.get(thSelector).should('have.class', '-sort-asc');
        // Do not assert because of potential timing problem: get td elements before table re-renders.
    });

    // TODO Investigate whether not yet supported or incorrect field in payload.
    it.skip('should sort the Top CVSS column', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        const thSelector = '.rt-th:contains("Top CVSS")';
        const tdSelector = '.rt-td:nth-child(4) [data-testid="label-chip"]';

        // 0. Initial table state indicates that the column is not sorted.
        cy.get(thSelector)
            .should('not.have.class', '-sort-asc')
            .should('not.have.class', '-sort-desc');

        // 1. Sort ascending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should('eq', '?sort[0][id]=Node%20Top%20CVSS&sort[0][desc]=false');

        cy.get(thSelector).should('have.class', '-sort-asc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfAscendingNumberValuesFromElements);
        });

        // 2. Sort descending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should('eq', '?sort[0][id]=Node%20Top%20CVSS&sort[0][desc]=true');

        cy.get(thSelector).should('have.class', '-sort-desc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingNumberValuesFromElements);
        });
    });

    it('should show a message when node scan data is incomplete', () => {
        const fixturePath = 'nodes/vmNodeOverview.json';

        cy.fixture(fixturePath).then((body) => {
            const { id } = body.data.result;
            const staticResponse = { body };
            visitVulnerabilityManagementEntityInSidePanel(entitiesKey, id, staticResponse);
        });

        cy.get(selectors.scanDataMessage);
    });
});
