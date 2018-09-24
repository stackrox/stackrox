import {
    url as violationsUrl,
    selectors as ViolationsPageSelectors
} from './constants/ViolationsPage';
import * as api from './constants/apiEndpoints';
import selectors from './constants/SearchPage';

describe('Violations page', () => {
    beforeEach(() => {
        cy.server();
        cy.fixture('alerts/alerts.json').as('alerts');
        cy.route('GET', api.alerts.alerts, '@alerts').as('alerts');
        cy.visit(violationsUrl);
        cy.wait('@alerts');
    });

    const mockGetAlert = () => {
        cy.fixture('alerts/alertById.json').as('alertById');
        cy.route('GET', api.alerts.alertById, '@alertById').as('alertById');
    };

    const mockGetAlertWithEmptyContainerConfig = () => {
        cy.fixture('alerts/alertWithEmptyContainerConfig.json').as('alertWithEmptyContainerConfig');
        cy
            .route('GET', api.alerts.alertById, '@alertWithEmptyContainerConfig')
            .as('alertWithEmptyContainerConfig');
    };

    it('should select item in nav bar', () => {
        cy.get(ViolationsPageSelectors.navLink).should('have.class', 'bg-primary-600');
    });

    it('should have violations in table', () => {
        cy.get(ViolationsPageSelectors.rows).should('have.length', 4);
    });

    it('should show the side panel on row click', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy
            .get(ViolationsPageSelectors.panels)
            .eq(1)
            .should('be.visible');
    });

    it('should show side panel with panel header', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstTableRow).click();
        cy.wait('@alertById');
        cy
            .get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.header)
            .should('have.text', 'tender_edison (z1137vn6nnmipffzpozr0f0ri)');
    });

    it('should have cluster column in table', () => {
        cy.get(ViolationsPageSelectors.clusterTableHeader).should('be.visible');
    });

    it('should close the side panel on search filter', () => {
        cy.visit(violationsUrl);
        cy.get(selectors.pageSearchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.pageSearchInput).type('remote{enter}', { force: true });
        cy
            .get(ViolationsPageSelectors.panels)
            .eq(1)
            .should('not.be.visible');
    });

    it('should have 3 tabs in the sidepanel', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy
            .get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .should('have.length', 3);
        cy
            .get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .eq(0)
            .should('have.text', 'Violations');
        cy
            .get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .eq(1)
            .should('have.text', 'Deployment Details');
        cy
            .get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .eq(2)
            .should('have.text', 'Policy Details');
    });

    it('should have a message in the Violations tab', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy
            .get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .get(ViolationsPageSelectors.sidePanel.getTabByIndex(0))
            .click();
        cy.get(ViolationsPageSelectors.collapsible.header).should('have.text', 'Violations');
        cy
            .get(ViolationsPageSelectors.collapsible.body)
            .contains(
                "Image name 'docker.io/library/redis:latest' matches the name policy 'tag=latest'"
            );
    });

    it('should have deployment information in the Deployment Details tab', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy
            .get(ViolationsPageSelectors.panels)
            .eq(1)
            .get(ViolationsPageSelectors.sidePanel.getTabByIndex(1))
            .click();
        cy.get(ViolationsPageSelectors.collapsible.header).should('have.length', 3);
        cy
            .get(ViolationsPageSelectors.collapsible.header)
            .eq(0)
            .should('have.text', 'Overview');
        cy
            .get(ViolationsPageSelectors.collapsible.header)
            .eq(1)
            .should('have.text', 'Container configuration');
        cy
            .get(ViolationsPageSelectors.collapsible.header)
            .eq(2)
            .should('have.text', 'Security Context');
    });

    it('should show deployment information in the Deployment Details tab with no container configuration values', () => {
        mockGetAlertWithEmptyContainerConfig();
        cy.get(ViolationsPageSelectors.lastTableRow).click();
        cy.wait('@alertWithEmptyContainerConfig');
        cy
            .get(ViolationsPageSelectors.panels)
            .eq(1)
            .get(ViolationsPageSelectors.sidePanel.getTabByIndex(1))
            .click();
        cy.get(ViolationsPageSelectors.securityBestPractices).should('not.have.text', 'Commands');
    });
});
