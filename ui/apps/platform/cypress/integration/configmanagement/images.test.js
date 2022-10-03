import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    entityListCountMatchesTableLinkCount,
    hasTabsFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from '../../helpers/configWorkflowUtils';
import { selectors as configManagementSelectors } from '../../constants/ConfigManagementPage';
import withAuth from '../../helpers/basicAuth';

// const entitiesKey = 'images'; // omit to minimize changed lines

describe('Config Management Entities (Images)', () => {
    withAuth();

    it('should render the images list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('images');
    });

    it('should open the side panel to show the same number of deployments when the deployments link is clicked', () => {
        entityListCountMatchesTableLinkCount('images', 'deployments', /^\d+ deployments?$/);
    });

    it('should take you to a images single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('images');
        navigateToSingleEntityPage('images');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('images');
        navigateToSingleEntityPage('images');
        hasCountWidgetsFor(['Deployments']);
    });

    it('should click on the deployments count widget in the entity page and show the deployments tab', () => {
        renderListAndSidePanel('images');
        navigateToSingleEntityPage('images');
        hasCountWidgetsFor(['Deployments']);
        clickOnCountWidget('deployments', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('images');
        navigateToSingleEntityPage('images');
        hasTabsFor(['deployments']);
    });

    it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
        const entitiesKey2 = 'deployments';

        context('Page', () => {
            renderListAndSidePanel('images');
            navigateToSingleEntityPage('images');
            pageEntityCountMatchesTableRows('images', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('images');
            sidePanelEntityCountMatchesTableRows('images', entitiesKey2);
        });
    });

    // regression test for ROX-4543-crash-when-drilling-down-to-image-deployments
    it('should allow user to drill down from cluster to image to image-deployments', () => {
        renderListAndSidePanel('clusters');
        clickOnCountWidget('images', 'side-panel');
        cy.get(`[data-testid="side-panel"] ${configManagementSelectors.tableRows}:last`).click({
            force: true,
        });
        clickOnCountWidget('deployments', 'entityList');

        // GraphQL error takes a while to show up, and just extending the cy.get timeout does not work with should-not
        cy.wait(1000);
        cy.get('[data-testid="graphql-error"]').should('not.exist');
    });
});
