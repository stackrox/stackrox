import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    clickOnSingularEntityWidgetInSidePanel,
    entityListCountMatchesTableLinkCount,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';

// const entitiesKey = 'deployments'; // omit to minimize changed lines

describe('Config Management Entities (Deployments)', () => {
    withAuth();

    it('should render the deployments list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('deployments');
    });

    it('should open the side panel to show the same number of secrets when the secrets link is clicked', () => {
        entityListCountMatchesTableLinkCount('deployments', 'secrets', /\d+ secrets?$/);
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        renderListAndSidePanel('deployments');
        clickOnSingularEntityWidgetInSidePanel('deployments', 'clusters');
    });

    it('should take you to a deployments single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('deployments');
        navigateToSingleEntityPage('deployments');
    });

    it('should show the related cluster, namespace, and service account widgets', () => {
        renderListAndSidePanel('deployments');
        navigateToSingleEntityPage('deployments');
        hasRelatedEntityFor('Cluster');
        hasRelatedEntityFor('Namespace');
        hasRelatedEntityFor('Service Account');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('deployments');
        navigateToSingleEntityPage('deployments');
        hasCountWidgetsFor(['Images', 'Secrets']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('deployments');
        navigateToSingleEntityPage('deployments');
        hasTabsFor(['images', 'secrets']);
    });

    it('should click on the images count widget in the entity page and show the images tab', () => {
        renderListAndSidePanel('deployments', 'collector');
        navigateToSingleEntityPage('deployments');
        clickOnCountWidget('images', 'entityList');
    });

    it('should have the same number of Images in the count widget as in the Images table', () => {
        const entitiesKey2 = 'images';

        context('Page', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployments');
            pageEntityCountMatchesTableRows('deployments', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('deployments');
            sidePanelEntityCountMatchesTableRows('deployments', entitiesKey2);
        });
    });

    it('should have the same number of Secrets in the count widget as in the Secrets table', () => {
        const entitiesKey2 = 'secrets';

        context('Page', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployments');
            pageEntityCountMatchesTableRows('deployments', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('deployments');
            sidePanelEntityCountMatchesTableRows('deployments', entitiesKey2);
        });
    });
});
