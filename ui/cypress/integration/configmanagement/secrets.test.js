import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    clickOnEntityWidget,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from '../../helpers/configWorkflowUtils';
import { url, selectors as configManagementSelectors } from '../../constants/ConfigManagementPage';
import withAuth from '../../helpers/basicAuth';

describe('Config Management Entities (Secrets)', () => {
    withAuth();

    it('should render the secrets list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('secrets');
    });

    it('should render the deployments link and open the side panel when a row is clicked', () => {
        cy.visit(url.list.secrets);
        cy.get(configManagementSelectors.tableRows)
            .find(`${configManagementSelectors.tableCells} a[data-testid='deployment']`)
            .eq(0)
            .click({ force: true })
            .invoke('text')
            .then((expectedText) => {
                cy.get('[data-testid="side-panel"] [data-testid="panel-header"]').contains(
                    expectedText.toLowerCase()
                );
            });
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        renderListAndSidePanel('secrets');
        clickOnEntityWidget('cluster', 'side-panel');
    });

    it('should take you to a secrets single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('secrets');
        navigateToSingleEntityPage('secret');
    });

    it('should show the related cluster widget', () => {
        renderListAndSidePanel('secrets');
        navigateToSingleEntityPage('secret');
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('secrets');
        navigateToSingleEntityPage('secret');
        hasCountWidgetsFor(['Deployments']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('secrets');
        navigateToSingleEntityPage('secret');
        hasTabsFor(['deployments']);
    });

    it('should click on the deployments count widget in the entity page and show the deployments tab', () => {
        renderListAndSidePanel('secrets');
        navigateToSingleEntityPage('secret');
        clickOnCountWidget('deployments', 'entityList');
    });

    it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
        context('Page', () => {
            renderListAndSidePanel('secrets');
            navigateToSingleEntityPage('secret');
            pageEntityCountMatchesTableRows('Deployments');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('secrets');
            sidePanelEntityCountMatchesTableRows('Deployments');
        });
    });
});
