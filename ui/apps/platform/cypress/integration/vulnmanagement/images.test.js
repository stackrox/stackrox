import withAuth from '../../helpers/basicAuth';
import {
    assertSortedItems,
    callbackForPairOfAscendingNumberValuesFromElements,
    callbackForPairOfDescendingNumberValuesFromElements,
} from '../../helpers/sort';
import {
    hasTableColumnHeadings,
    interactAndWaitForVulnerabilityManagementEntities,
    verifyConditionalCVEs,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
    visitVulnerabilityManagementEntityInSidePanel,
    visitVulnerabilityManagementSecondaryEntitiesInSidePanel,
} from './VulnerabilityManagement.helpers';
import { selectors } from './VulnerabilityManagement.selectors';

const entitiesKey = 'images';

describe('Vulnerability Management Images', () => {
    withAuth();

    it('should display table columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasTableColumnHeadings([
            '', // hidden
            'Image',
            'CVEs',
            'Top CVSS',
            'Created',
            'Scan Time',
            'Image OS',
            'Image Status',
            'Entities',
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
            '?sort[0][id]=Image%20Risk%20Priority&sort[0][desc]=true'
        );

        cy.get(thSelector).should('have.class', '-sort-desc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingNumberValuesFromElements);
        });

        // 2. Sort ascending by the column.
        cy.get(thSelector).click(); // no request because initial response has been cached
        cy.location('search').should(
            'eq',
            '?sort[0][id]=Image%20Risk%20Priority&sort[0][desc]=false'
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
        cy.location('search').should('eq', '?sort[0][id]=Image%20Top%20CVSS&sort[0][desc]=false');

        cy.get(thSelector).should('have.class', '-sort-asc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfAscendingNumberValuesFromElements);
        });

        // 2. Sort descending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should('eq', '?sort[0][id]=Image%20Top%20CVSS&sort[0][desc]=true');

        cy.get(thSelector).should('have.class', '-sort-desc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingNumberValuesFromElements);
        });
    });

    it('should show a message when image scan data is incomplete', () => {
        const fixturePath = 'images/vmImageOverview.json';

        cy.fixture(fixturePath).then((body) => {
            const { id } = body.data.result;
            const staticResponse = { body };
            visitVulnerabilityManagementEntityInSidePanel(entitiesKey, id, staticResponse);
        });

        cy.get(selectors.scanDataMessage);
    });

    // Argument 3 in verify functions is index of column which has the links.
    // The one-based index includes checkbox, hidden, invisible.

    it('should display either links for image CVEs or text for No CVEs', () => {
        verifyConditionalCVEs(entitiesKey, 'image-cves', 3, 'vulnCounter');
    });

    it('should display links for deployments', () => {
        verifySecondaryEntities(entitiesKey, 'deployments', 9);
    });

    it('should display links for image-components', () => {
        verifySecondaryEntities(entitiesKey, 'image-components', 9);
    });

    it('should show entity icon, not back button, if there is only one item on the side panel stack', () => {
        visitVulnerabilityManagementSecondaryEntitiesInSidePanel(entitiesKey, 'deployments', 9);

        cy.get(selectors.backButton).should('exist');
        cy.get(selectors.entityIcon).should('not.exist');

        cy.get(selectors.backButton).click();
        cy.get(selectors.backButton).should('not.exist');
        cy.get(selectors.entityIcon).should('exist');
    });
});
