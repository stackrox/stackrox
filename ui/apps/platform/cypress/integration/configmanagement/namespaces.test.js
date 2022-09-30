import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    clickOnSingularEntityWidgetInSidePanel,
    clickOnSingleEntityInTable,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
    entityListCountMatchesTableLinkCount,
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';

// const entitiesKey = 'namespaces'; // omit to minimize changed lines

describe('Config Management Entities (Namespaces)', () => {
    withAuth();

    it('should render the namespaces list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('namespaces');
    });

    it('should render the namespaces list and open the side panel with the clicked cluster value', () => {
        clickOnSingleEntityInTable('namespaces', 'clusters');
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        renderListAndSidePanel('namespaces');
        clickOnSingularEntityWidgetInSidePanel('namespaces', 'clusters');
    });

    it('should take you to a namespace single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('namespaces');
        navigateToSingleEntityPage('namespaces');
    });

    it('should show the related cluster widget', () => {
        renderListAndSidePanel('namespaces');
        navigateToSingleEntityPage('namespaces');
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('namespaces');
        navigateToSingleEntityPage('namespaces');
        hasCountWidgetsFor(['Deployments', 'Secrets', 'Images']);
    });

    it('should click on the secrets count widget in the entity page and show the secrets tab', () => {
        renderListAndSidePanel('namespaces', 'stackrox');
        navigateToSingleEntityPage('namespaces');
        clickOnCountWidget('secrets', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('namespaces');
        navigateToSingleEntityPage('namespaces');
        hasTabsFor(['deployments', 'secrets', 'images']);
    });

    it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
        const entitiesKey2 = 'deployments';

        context('Page', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespaces');
            pageEntityCountMatchesTableRows('namespaces', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('namespaces');
            sidePanelEntityCountMatchesTableRows('namespaces', entitiesKey2);
        });
    });

    it('should have the same number of Secrets in the count widget as in the Secrets table', () => {
        const entitiesKey2 = 'secrets';

        context('Page', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespaces');
            pageEntityCountMatchesTableRows('namespaces', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('namespaces');
            sidePanelEntityCountMatchesTableRows('namespaces', entitiesKey2);
        });
    });

    it('should have the same number of Images in the count widget as in the Images table', () => {
        const entitiesKey2 = 'images';

        context('Page', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespaces');
            pageEntityCountMatchesTableRows('namespaces', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('namespaces');
            sidePanelEntityCountMatchesTableRows('namespaces', entitiesKey2);
        });
    });

    it('should open the side panel to show the same number of Service Accounts when the Service Accounts link is clicked', () => {
        entityListCountMatchesTableLinkCount(
            'namespaces',
            'serviceaccounts',
            /^\d+ Service Accounts?$/
        );
    });

    it('should open the side panel to show the same number of Roles when the Roles link is clicked', () => {
        entityListCountMatchesTableLinkCount('namespaces', 'roles', /^\d+ Roles?$/);
    });
});
