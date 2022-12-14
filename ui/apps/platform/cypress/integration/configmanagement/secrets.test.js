import withAuth from '../../helpers/basicAuth';

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
} from './ConfigurationManagement.helpers';

const entitiesKey = 'secrets';

describe('Configuration Management Secrets', () => {
    withAuth();

    it('should render the secrets list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel(entitiesKey);
    });

    it('should render the deployments link and open the side panel when a row is clicked', () => {
        visitConfigurationManagementEntities(entitiesKey);

        cy.get('.rt-tbody .rt-tr')
            .find(`.rt-td a[data-testid='deployment']`)
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
        renderListAndSidePanel(entitiesKey);
        clickOnSingularEntityWidgetInSidePanel(entitiesKey, 'clusters');
    });

    it('should take you to a secrets single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
    });

    it('should show the related cluster widget', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasCountWidgetsFor(['Deployments']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['deployments']);
    });

    it('should click on the deployments count widget in the entity page and show the deployments tab', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        clickOnCountWidget('deployments', 'entityList');
    });

    describe('should have same number in deployments table as in count widget', () => {
        const entitiesKey2 = 'deployments';

        it('of page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });
});
