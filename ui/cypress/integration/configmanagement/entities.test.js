import { url, selectors } from '../constants/ConfigManagementPage';
import withAuth from '../helpers/basicAuth';

const renderListAndSidePanel = entity => {
    cy.visit(url.list[entity]);
    cy.get(selectors.tableRows)
        .eq(0)
        .find(selectors.tableCells)
        .eq(1)
        .click({ force: true });
    cy.get(selectors.widgets);
};

const navigateToSingleEntityPage = entity => {
    cy.get(selectors.externalLink).click();
    cy.url().should('contain', url.single[entity]);
};

const hasCountWidgetsFor = entities => {
    entities.forEach(entity => {
        cy.get(`${selectors.countWidgetTitle}:contains('${entity}')`);
    });
};

const hasTabsFor = entities => {
    entities.forEach(entity => {
        cy.get(selectors.groupedTabs)
            .find('div')
            .contains(entity);
    });
};

const hasRelatedEntityFor = entity => {
    cy.get(`${selectors.relatedEntityWidgetTitle}:contains('${entity}')`);
};

describe('Config Management Entities', () => {
    withAuth();

    context('Policy', () => {
        it('should render the policies list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('policies');
        });

        it('should take you to a policy single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('policies');
            navigateToSingleEntityPage('policy');
        });

        xit('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('policies');
            navigateToSingleEntityPage('policy');
            hasCountWidgetsFor(['Deployments']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('policies');
            navigateToSingleEntityPage('policy');
            hasTabsFor(['deployments']);
        });
    });

    context('CIS Control', () => {
        it('should render the controls list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('controls');
        });

        it('should take you to a control single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('controls');
            navigateToSingleEntityPage('control');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('controls');
            navigateToSingleEntityPage('control');
            hasCountWidgetsFor(['Nodes']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('controls');
            navigateToSingleEntityPage('control');
            hasTabsFor(['nodes']);
        });
    });

    context('Cluster', () => {
        it('should render the clusters list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('clusters');
        });

        it('should take you to a cluster single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
        });

        xit('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            hasCountWidgetsFor([
                'Nodes',
                'Namespaces',
                'Deployments',
                'Images',
                'Secrets',
                'Users & Groups',
                'Service Accounts',
                'Roles',
                'Policies',
                'Controls'
            ]);
        });

        xit('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            hasTabsFor([
                'nodes',
                'namespaces',
                'deployments',
                'images',
                'secrets',
                'users and groups',
                'service accounts',
                'roles',
                'policies',
                'controls'
            ]);
        });
    });

    context('Namespace', () => {
        it('should render the namespaces list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('namespaces');
        });

        it('should take you to a namespace single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespace');
        });

        xit('should show the related cluster widget', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namspace');
            hasRelatedEntityFor('cluster');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespace');
            hasCountWidgetsFor(['Deployments', 'Secrets', 'Images', 'Policies']);
        });

        xit('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespace');
            hasTabsFor(['deployments', 'secrets', 'images', 'policies']);
        });
    });

    context('Node', () => {
        it('should render the nodes list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('nodes');
        });

        it('should take you to a nodes single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('nodes');
            navigateToSingleEntityPage('node');
        });

        xit('should show the related cluster widget', () => {
            renderListAndSidePanel('nodes');
            navigateToSingleEntityPage('node');
            hasRelatedEntityFor('cluster');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('nodes');
            navigateToSingleEntityPage('node');
            hasCountWidgetsFor(['Controls']);
        });

        xit('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('nodes');
            navigateToSingleEntityPage('node');
            hasTabsFor(['controls']);
        });
    });

    context('Deployment', () => {
        it('should render the deployments list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('deployments');
        });

        it('should take you to a deployments single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployment');
        });

        xit('should show the related cluster, namespace, and service account widgets', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployment');
            hasRelatedEntityFor('cluster');
            hasRelatedEntityFor('namespace');
            hasRelatedEntityFor('service account');
        });

        xit('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployment');
            hasCountWidgetsFor(['Images', 'Policies']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployment');
            hasTabsFor(['images', 'policies']);
        });
    });

    context('Image', () => {
        it('should render the images list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('images');
        });

        it('should take you to a images single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('images');
            navigateToSingleEntityPage('image');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('images');
            navigateToSingleEntityPage('image');
            hasCountWidgetsFor(['Deployments']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('images');
            navigateToSingleEntityPage('image');
            hasTabsFor(['deployments']);
        });
    });

    xcontext('Secret', () => {
        it('should render the secrets list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('secrets');
        });

        it('should take you to a secrets single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('secrets');
            navigateToSingleEntityPage('secret');
        });

        it('should show the related namespace widget', () => {
            renderListAndSidePanel('secrets');
            navigateToSingleEntityPage('secret');
            hasRelatedEntityFor('namespace');
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
    });

    xcontext('Role', () => {
        it('should render the roles list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('roles');
        });

        it('should take you to a roles single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('roles');
            navigateToSingleEntityPage('role');
        });

        it('should show the related namespace scope widget', () => {
            renderListAndSidePanel('roles');
            navigateToSingleEntityPage('role');
            hasRelatedEntityFor('namespace scope');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('roles');
            navigateToSingleEntityPage('role');
            hasCountWidgetsFor(['Users & Groups', 'Service Accounts']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('roles');
            navigateToSingleEntityPage('role');
            hasTabsFor(['users and groups', 'service accounts']);
        });
    });

    context('Subject (Users & Groups)', () => {
        it('should render the users & groups list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('subjects');
        });

        it('should take you to a subject single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('subjects');
            navigateToSingleEntityPage('subject');
        });

        xit('should show the related cluster widget', () => {
            renderListAndSidePanel('subjects');
            navigateToSingleEntityPage('subject');
            hasRelatedEntityFor('cluster');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('subjects');
            navigateToSingleEntityPage('subject');
            hasCountWidgetsFor(['Roles']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('subjects');
            navigateToSingleEntityPage('subject');
            hasTabsFor(['roles']);
        });
    });

    context('Service Account', () => {
        it('should render the service accounts list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('serviceAccounts');
        });

        it('should take you to a service account single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('serviceAccounts');
            navigateToSingleEntityPage('serviceAccount');
        });

        xit('should show the related namespace widget', () => {
            renderListAndSidePanel('serviceAccounts');
            navigateToSingleEntityPage('serviceAccount');
            hasRelatedEntityFor('namespace');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('serviceAccounts');
            navigateToSingleEntityPage('serviceAccount');
            hasCountWidgetsFor(['Deployments', 'Secrets', 'Roles']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('serviceAccounts');
            navigateToSingleEntityPage('serviceAccount');
            hasTabsFor(['deployments', 'secrets', 'roles']);
        });
    });
});
