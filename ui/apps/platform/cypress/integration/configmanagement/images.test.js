import withAuth from '../../helpers/basicAuth';

import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    entityListCountMatchesTableLinkCount,
    hasTabsFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from './ConfigurationManagement.helpers';

const entitiesKey = 'images';

describe('Configuration Management Images', () => {
    withAuth();

    it('should render the images list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel(entitiesKey);
    });

    it('should open the side panel to show the same number of deployments when the deployments link is clicked', () => {
        entityListCountMatchesTableLinkCount(entitiesKey, 'deployments', /^\d+ deployments?$/);
    });

    it('should take you to a images single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasCountWidgetsFor(['Deployments']);
    });

    it('should click on the deployments count widget in the entity page and show the deployments tab', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasCountWidgetsFor(['Deployments']);
        clickOnCountWidget('deployments', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['deployments']);
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

    // regression test for ROX-4543-crash-when-drilling-down-to-image-deployments
    it('should allow user to drill down from cluster to image to image-deployments', () => {
        renderListAndSidePanel('clusters');
        clickOnCountWidget(entitiesKey, 'side-panel');
        cy.get(`[data-testid="side-panel"] .rt-tbody .rt-tr:last`).click({
            force: true,
        });
        clickOnCountWidget('deployments', 'entityList');

        // GraphQL error takes a while to show up, and just extending the cy.get timeout does not work with should-not
        cy.wait(1000);
        cy.get('[data-testid="graphql-error"]').should('not.exist');
    });
});
