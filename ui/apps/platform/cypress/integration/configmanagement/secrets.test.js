import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    clickOnSingularEntityWidgetInSidePanel,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
    visitConfigurationManagementEntities,
} from '../../helpers/configWorkflowUtils';
import { selectors as configManagementSelectors } from '../../constants/ConfigManagementPage';
import withAuth from '../../helpers/basicAuth';

// const entitiesKey = 'secrets'; // omit to minimize changed lines

describe('Config Management Entities (Secrets)', () => {
    withAuth();

    it('should render the secrets list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('secrets');
    });

    it('should render the deployments link and open the side panel when a row is clicked', () => {
        visitConfigurationManagementEntities('secrets');

        cy.get(configManagementSelectors.tableRows)
            .find(`${configManagementSelectors.tableCells} a[data-testid='deployment']`)
            .eq(0)
            .click()
            .invoke('text')
            .then((expectedText) => {
                cy.get('[data-testid="side-panel"] [data-testid="panel-header"]').contains(
                    expectedText.toLowerCase()
                );
            });
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        renderListAndSidePanel('secrets');
        clickOnSingularEntityWidgetInSidePanel('clusters');
    });

    it('should take you to a secrets single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('secrets');
        navigateToSingleEntityPage('secrets');
    });

    it('should show the related cluster widget', () => {
        renderListAndSidePanel('secrets');
        navigateToSingleEntityPage('secrets');
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('secrets');
        navigateToSingleEntityPage('secrets');
        hasCountWidgetsFor(['Deployments']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('secrets');
        navigateToSingleEntityPage('secrets');
        hasTabsFor(['deployments']);
    });

    it('should click on the deployments count widget in the entity page and show the deployments tab', () => {
        renderListAndSidePanel('secrets');
        navigateToSingleEntityPage('secrets');
        clickOnCountWidget('deployments', 'entityList');
    });

    it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
        const entitiesKey2 = 'deployments';

        context('Page', () => {
            renderListAndSidePanel('secrets');
            navigateToSingleEntityPage('secrets');
            pageEntityCountMatchesTableRows('secrets', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('secrets');
            sidePanelEntityCountMatchesTableRows('secrets', entitiesKey2);
        });
    });
});
